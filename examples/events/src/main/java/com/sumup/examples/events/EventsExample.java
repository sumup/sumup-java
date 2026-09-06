package com.sumup.examples.events;

import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.events.EventCallbackException;
import com.sumup.sdk.events.EventPayloadException;
import com.sumup.sdk.events.EventSignature;
import com.sumup.sdk.events.EventSignatureException;
import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;

/** A minimal event receiver using the JDK HTTP server. */
public final class EventsExample {
  private EventsExample() {}

  /** Starts a receiver on port 8080. Set SUMUP_API_KEY and SUMUP_EVENT_SECRET first. */
  public static void main(String[] args) throws Exception {
    var client = new SumUpClient();
    var events =
        client.eventsHandler(
            System.getenv("SUMUP_EVENT_SECRET"),
            event -> System.out.printf("Unhandled event %s: %s%n", event.id(), event.type()));
    events.onMemberUpdated(
        event -> {
          var member = event.fetchObject();
          System.out.printf("Member updated: %s%n", member.id());
        });
    var server = HttpServer.create(new InetSocketAddress(8080), 0);
    server.createContext(
        "/events",
        request -> {
          try {
            if (!"POST".equals(request.getRequestMethod())) {
              request.sendResponseHeaders(405, -1);
              return;
            }
            var body = request.getRequestBody().readAllBytes();
            try {
              events.handle(body, request.getRequestHeaders().getFirst(EventSignature.HEADER_NAME));
              request.sendResponseHeaders(204, -1);
            } catch (EventSignatureException | EventPayloadException failure) {
              request.sendResponseHeaders(400, -1);
            } catch (EventCallbackException failure) {
              failure.printStackTrace();
              request.sendResponseHeaders(500, -1);
            }
          } finally {
            request.close();
          }
        });
    server.start();
    System.out.println("Listening on http://localhost:8080/events");
  }
}
