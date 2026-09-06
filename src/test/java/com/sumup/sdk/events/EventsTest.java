package com.sumup.sdk.events;

import static org.junit.jupiter.api.Assertions.*;

import com.sumup.sdk.SumUpAsyncClient;
import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.core.ApiException;
import com.sumup.sdk.core.RequestOptions;
import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.HexFormat;
import java.util.concurrent.CancellationException;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionException;
import java.util.concurrent.atomic.AtomicInteger;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

final class EventsTest {
  private static final String SECRET = "test_secret";
  private static final long NOW = 1788600000;
  private static final SumUpClient CLIENT = new SumUpClient("test_key");

  private static byte[] bytes(String body) {
    return body.getBytes(StandardCharsets.UTF_8);
  }

  private static byte[] body(String type) {
    return bytes(
        "{\"id\":\"evt_123\",\"type\":\""
            + type
            + "\",\"created_at\":\"2026-09-05T09:30:00Z\",\"object\":{\"id\":\"123\",\"type\":\"member\"}}");
  }

  private static String sign(byte[] body) throws Exception {
    return sign(body, Long.toString(Instant.now().getEpochSecond()));
  }

  private static String sign(byte[] body, String timestamp) throws Exception {
    var mac = Mac.getInstance("HmacSHA256");
    mac.init(new SecretKeySpec(bytes(SECRET), "HmacSHA256"));
    mac.update(bytes("v1:" + timestamp + ":"));
    return "t=" + timestamp + ",v1=" + HexFormat.of().formatHex(mac.doFinal(body));
  }

  @ParameterizedTest
  @ValueSource(longs = {-300, 0, 300})
  void verifiesBoundariesAndTimestampText(long offset) throws Exception {
    var body = bytes("héllo");
    var signature = sign(body, "00" + (NOW + offset));
    EventSignature.verify(body, "  " + signature + "  ", SECRET, NOW);
    EventSignature.verify(
        body,
        signature.substring(0, signature.indexOf("v1="))
            + "v1="
            + signature.substring(signature.indexOf("v1=") + 3).toUpperCase(),
        SECRET,
        NOW);
  }

  @ParameterizedTest
  @ValueSource(longs = {-301, 301})
  void rejectsExpiredSignatures(long offset) throws Exception {
    var body = body("members.created");
    var signature = sign(body, Long.toString(NOW + offset));
    assertThrows(
        EventSignatureExpiredException.class,
        () -> EventSignature.verify(body, signature, SECRET, NOW));
  }

  @ParameterizedTest
  @ValueSource(
      strings = {
        "",
        "v1=ab",
        "t=1,v1=ab",
        "t=-1,v1=ab",
        "t=+1,v1=ab",
        "t=1, v1=ab",
        "t=1,v1=ab,t=1",
        "v1=ab,t=1"
      })
  void rejectsMalformedHeaders(String header) {
    assertThrows(
        EventSignatureException.class,
        () -> EventSignature.verify(bytes("{}"), header, SECRET, NOW));
  }

  @Test
  void rejectsMissingHeadersChangedBytesAndWrongSecrets() throws Exception {
    var body = body("members.created");
    var signature = sign(body);
    assertThrows(EventSignatureException.class, () -> EventSignature.verify(body, null, SECRET));
    assertThrows(
        EventSignatureException.class, () -> EventSignature.verify(body, signature, "other"));
    body[0] ^= 1;
    assertThrows(
        EventSignatureException.class, () -> EventSignature.verify(body, signature, SECRET));
    assertThrows(IllegalArgumentException.class, () -> EventSignature.verify(body, signature, ""));
    var overflow = sign(body, "9223372036854775808");
    assertThrows(
        EventSignatureException.class, () -> EventSignature.verify(body, overflow, SECRET));
    var future = sign(body, Long.toString(Long.MAX_VALUE));
    assertThrows(
        EventSignatureExpiredException.class, () -> EventSignature.verify(body, future, SECRET));
  }

  @Test
  void parsesAllKnownTypesAndUnknownEvents() throws Exception {
    String[] names = {
      "members.created",
      "members.updated",
      "members.deleted",
      "readers.created",
      "readers.deleted",
      "future.event"
    };
    Class<?>[] types = {
      MemberCreatedEvent.class,
      MemberUpdatedEvent.class,
      MemberDeletedEvent.class,
      ReaderCreatedEvent.class,
      ReaderDeletedEvent.class,
      EventNotification.class
    };
    for (int i = 0; i < names.length; i++) {
      var body = body(names[i]);
      var event = CLIENT.parseEventNotification(body, sign(body), SECRET);
      assertEquals(types[i], event.getClass());
      assertEquals(names[i], event.type());
      assertEquals("evt_123", event.id());
      assertEquals(2026, event.createdAt().getYear());
      assertEquals(types[i], CLIENT.parseEventNotificationWithoutVerification(body).getClass());
    }
    assertNotNull(CLIENT.parseEventNotificationWithoutVerification(bytes("{}")));
    assertInstanceOf(
        MemberUpdatedEvent.class,
        CLIENT.parseEventNotificationWithoutVerification(
            bytes("{\"type\":\"members.updated\",\"object\":{\"type\":\"future\"}}")));
  }

  @ParameterizedTest
  @ValueSource(strings = {"null", "[]", "{} {}", "{", "{\"created_at\":\"invalid\"}"})
  void reportsInvalidJson(String json) {
    assertThrows(
        EventPayloadException.class,
        () -> CLIENT.parseEventNotificationWithoutVerification(bytes(json)));
    assertThrows(
        EventSignatureException.class,
        () -> CLIENT.parseEventNotification(bytes(json), null, SECRET));
  }

  @Test
  void dispatchesReplacesAndFallsBack() throws Exception {
    var calls = new AtomicInteger();
    var fallback = new AtomicInteger();
    var handler = CLIENT.eventsHandler(SECRET, event -> fallback.incrementAndGet());
    handler.onMemberUpdated(
        event -> {
          fail("replaced callback");
        });
    handler.onMemberUpdated(event -> calls.incrementAndGet());
    for (var type : new String[] {"members.updated", "members.created", "future.event"}) {
      var body = body(type);
      handler.handle(body, sign(body));
    }
    assertEquals(1, calls.get());
    assertEquals(2, fallback.get());
    var cause = new Exception("failure");
    handler.onMemberUpdated(
        event -> {
          throw cause;
        });
    var body = body("members.updated");
    var signature = sign(body);
    assertSame(
        cause,
        assertThrows(EventCallbackException.class, () -> handler.handle(body, signature))
            .getCause());
  }

  @Test
  void asyncHandlingWaitsAndPreservesFailures() throws Exception {
    var completion = new CompletableFuture<Void>();
    var handler =
        new SumUpAsyncClient("key")
            .eventsHandler(SECRET, event -> CompletableFuture.completedFuture(null));
    handler.onMemberUpdated(event -> completion);
    var body = body("members.updated");
    var result = handler.handleAsync(body, sign(body));
    assertFalse(result.isDone());
    completion.complete(null);
    result.join();
    var cause = new IllegalStateException("failure");
    handler.onMemberUpdated(event -> CompletableFuture.failedFuture(cause));
    var failed = handler.handleAsync(body, sign(body));
    assertSame(
        cause,
        assertInstanceOf(
                EventCallbackException.class,
                assertThrows(CompletionException.class, failed::join).getCause())
            .getCause());
    handler.onMemberUpdated(
        event -> {
          throw cause;
        });
    assertInstanceOf(
        EventCallbackException.class,
        assertThrows(CompletionException.class, () -> handler.handleAsync(body, sign(body)).join())
            .getCause());
    handler.onMemberUpdated(event -> CompletableFuture.failedFuture(new CancellationException()));
    var cancelled = handler.handleAsync(body, sign(body));
    assertInstanceOf(
        CancellationException.class,
        assertThrows(CompletionException.class, cancelled::join).getCause());
  }

  @ParameterizedTest
  @ValueSource(
      strings = {
        "https://evil.example/member",
        "http://api.sumup.com/member",
        "https://api.sumup.com:444/member",
        "/member",
        "file:///member",
        "https://api.sumup.com.evil.example/member",
        "https://api.sumup.com//evil.example/member"
      })
  void rejectsForeignResourceOrigins(String url) {
    var event = resource(CLIENT, url);
    assertThrows(EventObjectException.class, event::fetchObject);
    assertThrows(EventObjectException.class, event::fetchObjectAsync);
  }

  @Test
  void rejectsRedirectFollowingTransports() {
    var client =
        SumUpClient.builder()
            .httpClient(
                java.net.http.HttpClient.newBuilder()
                    .followRedirects(java.net.http.HttpClient.Redirect.ALWAYS)
                    .build())
            .build();
    assertThrows(
        EventObjectException.class,
        () -> resource(client, "https://api.sumup.com/member").fetchObject());
  }

  private static MemberUpdatedEvent resource(SumUpClient client, String url) {
    return (MemberUpdatedEvent)
        client.parseEventNotificationWithoutVerification(
            bytes("{\"type\":\"members.updated\",\"object\":{\"url\":\"" + url + "\"}}"));
  }

  @Test
  void fetchesUsingConfiguredTransportAndPreservesEscapes() throws Exception {
    var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
    var requests = new AtomicInteger();
    server.createContext(
        "/",
        exchange -> {
          try {
            assertEquals(
                requests.get() == 0 ? "Bearer key" : "Bearer override",
                exchange.getRequestHeaders().getFirst("Authorization"));
            assertEquals("/member%2F123?expand=a%2Fb", exchange.getRequestURI().toString());
            requests.incrementAndGet();
            var payload = bytes("{\"id\":\"123\"}");
            exchange.sendResponseHeaders(200, payload.length);
            exchange.getResponseBody().write(payload);
          } finally {
            exchange.close();
          }
        });
    server.createContext(
        "/missing",
        exchange -> {
          exchange.sendResponseHeaders(404, -1);
          exchange.close();
        });
    server.start();
    try {
      var origin = "http://127.0.0.1:" + server.getAddress().getPort();
      var client = SumUpClient.builder().accessToken("key").baseUri(origin).build();
      var event =
          resource(
              client,
              origin.replace("http://", "http://user:pass@")
                  + "/member%2F123?expand=a%2Fb#ignored");
      assertEquals("123", event.fetchObject().id());
      assertEquals(
          "123",
          event
              .fetchObjectAsync(
                  RequestOptions.builder().authorizationHeader("Bearer override").build())
              .join()
              .id());
      assertEquals(2, requests.get());
      assertEquals(
          404,
          assertThrows(
                  ApiException.class, () -> resource(client, origin + "/missing").fetchObject())
              .getStatusCode());
      var failure =
          assertThrows(
              CompletionException.class,
              () -> resource(client, origin + "/missing").fetchObjectAsync().join());
      assertInstanceOf(ApiException.class, failure.getCause());
    } finally {
      server.stop(0);
    }
  }
}
