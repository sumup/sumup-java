package com.sumup.sdk.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertSame;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.fasterxml.jackson.core.type.TypeReference;
import com.sumup.sdk.clients.ReadersClient;
import java.io.IOException;
import java.net.Authenticator;
import java.net.CookieHandler;
import java.net.ProxySelector;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpHeaders;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executor;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLParameters;
import javax.net.ssl.SSLSession;
import org.junit.jupiter.api.Test;

final class ApiClientTest {

  @Test
  void defaultAuthorizationUsesAccessToken() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient client = ApiClient.builder().httpClient(httpClient).accessToken("token").build();

    client.send(HttpMethod.GET, "/v1/test", null, null, null, null, null);

    assertEquals(
        "Bearer token",
        httpClient.lastRequest().headers().firstValue("Authorization").orElse(null));
  }

  @Test
  void requestOptionsAuthorizationOverridesAccessToken() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient client = ApiClient.builder().httpClient(httpClient).accessToken("token").build();
    RequestOptions requestOptions =
        RequestOptions.builder().authorizationHeader("Basic custom").build();

    client.send(HttpMethod.GET, "/v1/test", null, null, null, null, requestOptions);

    assertEquals(
        "Basic custom",
        httpClient.lastRequest().headers().firstValue("Authorization").orElse(null));
  }

  @Test
  void defaultUserAgentIncludesSdkVersion() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient client = ApiClient.builder().httpClient(httpClient).build();

    client.send(HttpMethod.GET, "/v1/test", null, null, null, null, null);

    assertEquals(
        SdkMetadata.userAgent(),
        httpClient.lastRequest().headers().firstValue("User-Agent").orElse(null));
  }

  @Test
  void requestOptionsCanOverrideUserAgent() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient client = ApiClient.builder().httpClient(httpClient).build();
    RequestOptions requestOptions =
        RequestOptions.builder().header("User-Agent", "custom/agent").build();

    client.send(HttpMethod.GET, "/v1/test", null, null, null, null, requestOptions);

    assertEquals(
        "custom/agent", httpClient.lastRequest().headers().firstValue("User-Agent").orElse(null));
  }

  @Test
  void defaultRuntimeHeadersAreIncluded() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient client = ApiClient.builder().httpClient(httpClient).build();

    client.send(HttpMethod.GET, "/v1/test", null, null, null, null, null);

    HttpHeaders headers = httpClient.lastRequest().headers();
    assertEquals(SdkMetadata.apiVersion(), headers.firstValue("X-Sumup-Api-Version").orElse(null));
    assertEquals("java", headers.firstValue("X-Sumup-Lang").orElse(null));
    assertEquals(SdkMetadata.version(), headers.firstValue("X-Sumup-Package-Version").orElse(null));
    assertEquals(
        System.getProperty("os.name", "unknown"), headers.firstValue("X-Sumup-OS").orElse(null));
    assertEquals(
        SdkMetadata.runtimeHeaders().get("X-Sumup-Arch"),
        headers.firstValue("X-Sumup-Arch").orElse(null));
    assertEquals("java", headers.firstValue("X-Sumup-Runtime").orElse(null));
    assertEquals(
        Runtime.version().toString(), headers.firstValue("X-Sumup-Runtime-Version").orElse(null));
    assertEquals(
        "application/problem+json, application/json", headers.firstValue("Accept").orElse(null));
  }

  @Test
  void requestOptionsCanOverrideTimeout() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient client = ApiClient.builder().httpClient(httpClient).build();
    Duration timeout = Duration.ofSeconds(5);
    RequestOptions requestOptions = RequestOptions.builder().timeout(timeout).build();

    client.send(HttpMethod.GET, "/v1/test", null, null, null, null, requestOptions);

    assertTrue(httpClient.lastRequest().timeout().isPresent());
    assertEquals(timeout, httpClient.lastRequest().timeout().get());
  }

  @Test
  void readerPathParamsEncodeStrings() {
    CapturingHttpClient httpClient = new CapturingHttpClient();
    ApiClient apiClient = ApiClient.builder().httpClient(httpClient).build();
    ReadersClient readersClient = new ReadersClient(apiClient);

    readersClient.delete("merchant-code", "reader 123");

    assertEquals(
        URI.create("https://api.sumup.com/v0.1/merchants/merchant-code/readers/reader+123"),
        httpClient.lastRequest().uri());
  }

  @Test
  void apiErrorsExposeTypedProblemsAndKeepResponseContext() {
    String body = "{\"type\":\"about:blank\",\"detail\":\"Invalid amount\"}";
    ApiClient client = ApiClient.builder().httpClient(new CapturingHttpClient(400, body)).build();
    ApiStatusException error =
        assertThrows(
            ApiStatusException.class,
            () -> client.send(HttpMethod.GET, "/error", null, null, null, null, null));
    assertEquals("Invalid amount", error.getProblem().orElseThrow().detail());
    assertEquals(body, error.getResponseBody());
    assertEquals(400, error.getStatusCode());
    assertEquals("10", error.getResponseHeaders().get("Retry-After").get(0));
    java.util.concurrent.CompletionException asyncError =
        assertThrows(
            java.util.concurrent.CompletionException.class,
            () -> client.sendAsync(HttpMethod.GET, "/error", null, null, null, null, null).join());
    assertInstanceOf(ApiStatusException.class, asyncError.getCause());
  }

  @Test
  void problemExtensionsCanContainNull() {
    String body = "{\"type\":\"about:blank\",\"detail\":\"Token expired\",\"trace_id\":null}";
    ApiClient client = ApiClient.builder().httpClient(new CapturingHttpClient(401, body)).build();
    ApiStatusException error =
        assertThrows(
            ApiStatusException.class,
            () -> client.send(HttpMethod.GET, "/error", null, null, null, null, null));
    var problem = error.getProblem().orElseThrow();
    assertEquals("Token expired", problem.detail());
    assertTrue(problem.additionalProperties().containsKey("trace_id"));
    assertNull(problem.additionalProperties().get("trace_id"));
    assertThrows(
        UnsupportedOperationException.class,
        () -> problem.additionalProperties().put("trace_id", "changed"));
  }

  @Test
  void problemTitleIsUsedWhenDetailIsMissing() {
    String body = "{\"type\":\"about:blank\",\"title\":\"Unauthorized\"}";
    ApiClient client = ApiClient.builder().httpClient(new CapturingHttpClient(401, body)).build();
    ApiStatusException error =
        assertThrows(
            ApiStatusException.class,
            () -> client.send(HttpMethod.GET, "/error", null, null, null, null, null));
    assertEquals("Unauthorized (401)", error.getMessage());
  }

  @Test
  void malformedErrorBodiesRemainHttpErrors() {
    ApiClient client =
        ApiClient.builder().httpClient(new CapturingHttpClient(502, "{invalid")).build();
    ApiStatusException error =
        assertThrows(
            ApiStatusException.class,
            () -> client.send(HttpMethod.GET, "/error", null, null, null, null, null));
    assertTrue(error.getProblem().isEmpty());
    assertEquals("{invalid", error.getResponseBody());
    assertEquals(502, error.getStatusCode());
  }

  @Test
  void malformedSuccessIsNotAConnectionFailure() {
    ApiClient client =
        ApiClient.builder().httpClient(new CapturingHttpClient(200, "{invalid")).build();
    var type = new TypeReference<Map<String, Object>>() {};
    ApiException error =
        assertThrows(
            ApiException.class,
            () -> client.send(HttpMethod.GET, "/test", null, null, null, type, null));
    assertFalse(error instanceof ApiConnectionException);
    assertEquals("Failed to decode response", error.getMessage());
    var asyncError =
        assertThrows(
            java.util.concurrent.CompletionException.class,
            () -> client.sendAsync(HttpMethod.GET, "/test", null, null, null, type, null).join());
    assertEquals(ApiException.class, asyncError.getCause().getClass());
    assertEquals(error.getMessage(), asyncError.getCause().getMessage());
  }

  @Test
  void transportFailuresPreserveTheirTypeAndCause() {
    for (IOException cause :
        new IOException[] {
          new IOException("Disconnected"), new java.net.http.HttpTimeoutException("Deadline")
        }) {
      ApiClient client = ApiClient.builder().httpClient(new CapturingHttpClient(cause)).build();
      var error =
          assertThrows(
              ApiConnectionException.class,
              () -> client.send(HttpMethod.GET, "/test", null, null, null, null, null));
      assertSame(cause, error.getCause());
      assertEquals(
          cause instanceof java.net.http.HttpTimeoutException,
          error instanceof ApiTimeoutException);
      var asyncError =
          assertThrows(
              java.util.concurrent.CompletionException.class,
              () -> client.sendAsync(HttpMethod.GET, "/test", null, null, null, null, null).join());
      assertEquals(error.getClass(), asyncError.getCause().getClass());
      assertSame(cause, asyncError.getCause().getCause());
    }
  }

  private static final class CapturingHttpClient extends HttpClient {
    private HttpRequest lastRequest;
    private final int status;
    private final String body;
    private final IOException failure;

    CapturingHttpClient() {
      this(200, "");
    }

    CapturingHttpClient(int status, String body) {
      this.status = status;
      this.body = body;
      this.failure = null;
    }

    CapturingHttpClient(IOException failure) {
      this.status = 0;
      this.body = "";
      this.failure = failure;
    }

    HttpRequest lastRequest() {
      return lastRequest;
    }

    @Override
    public Optional<CookieHandler> cookieHandler() {
      return Optional.empty();
    }

    @Override
    public Optional<Duration> connectTimeout() {
      return Optional.empty();
    }

    @Override
    public Redirect followRedirects() {
      return Redirect.NEVER;
    }

    @Override
    public Optional<ProxySelector> proxy() {
      return Optional.empty();
    }

    @Override
    public SSLContext sslContext() {
      return null;
    }

    @Override
    public SSLParameters sslParameters() {
      return new SSLParameters();
    }

    @Override
    public Optional<Authenticator> authenticator() {
      return Optional.empty();
    }

    @Override
    public HttpClient.Version version() {
      return HttpClient.Version.HTTP_1_1;
    }

    @Override
    public Optional<Executor> executor() {
      return Optional.empty();
    }

    @Override
    public <T> HttpResponse<T> send(
        HttpRequest request, HttpResponse.BodyHandler<T> responseBodyHandler) throws IOException {
      this.lastRequest = request;
      if (failure != null) throw failure;
      return (HttpResponse<T>) new CapturingHttpResponse(request, status, body);
    }

    @Override
    public <T> CompletableFuture<HttpResponse<T>> sendAsync(
        HttpRequest request, HttpResponse.BodyHandler<T> responseBodyHandler) {
      this.lastRequest = request;
      if (failure != null) return CompletableFuture.failedFuture(failure);
      return CompletableFuture.completedFuture(
          (HttpResponse<T>) new CapturingHttpResponse(request, status, body));
    }

    @Override
    public <T> CompletableFuture<HttpResponse<T>> sendAsync(
        HttpRequest request,
        HttpResponse.BodyHandler<T> responseBodyHandler,
        HttpResponse.PushPromiseHandler<T> pushPromiseHandler) {
      this.lastRequest = request;
      if (failure != null) return CompletableFuture.failedFuture(failure);
      return CompletableFuture.completedFuture(
          (HttpResponse<T>) new CapturingHttpResponse(request, status, body));
    }
  }

  private static final class CapturingHttpResponse implements HttpResponse<String> {
    private final HttpRequest request;
    private final int status;
    private final String body;

    CapturingHttpResponse(HttpRequest request, int status, String body) {
      this.request = request;
      this.status = status;
      this.body = body;
    }

    @Override
    public int statusCode() {
      return status;
    }

    @Override
    public HttpRequest request() {
      return request;
    }

    @Override
    public Optional<HttpResponse<String>> previousResponse() {
      return Optional.empty();
    }

    @Override
    public HttpHeaders headers() {
      return HttpHeaders.of(Map.of("Retry-After", java.util.List.of("10")), (a, b) -> true);
    }

    @Override
    public String body() {
      return body;
    }

    @Override
    public Optional<SSLSession> sslSession() {
      return Optional.empty();
    }

    @Override
    public URI uri() {
      return request.uri();
    }

    @Override
    public HttpClient.Version version() {
      return HttpClient.Version.HTTP_1_1;
    }
  }
}
