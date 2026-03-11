package com.sumup.sdk.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

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
    assertEquals("application/problem+json, application/json", headers.firstValue("Accept").orElse(null));
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

  private static final class CapturingHttpClient extends HttpClient {
    private HttpRequest lastRequest;

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
        HttpRequest request, HttpResponse.BodyHandler<T> responseBodyHandler) {
      this.lastRequest = request;
      return (HttpResponse<T>) new CapturingHttpResponse(request);
    }

    @Override
    public <T> CompletableFuture<HttpResponse<T>> sendAsync(
        HttpRequest request, HttpResponse.BodyHandler<T> responseBodyHandler) {
      this.lastRequest = request;
      return CompletableFuture.completedFuture(
          (HttpResponse<T>) new CapturingHttpResponse(request));
    }

    @Override
    public <T> CompletableFuture<HttpResponse<T>> sendAsync(
        HttpRequest request,
        HttpResponse.BodyHandler<T> responseBodyHandler,
        HttpResponse.PushPromiseHandler<T> pushPromiseHandler) {
      this.lastRequest = request;
      return CompletableFuture.completedFuture(
          (HttpResponse<T>) new CapturingHttpResponse(request));
    }
  }

  private static final class CapturingHttpResponse implements HttpResponse<String> {
    private final HttpRequest request;

    CapturingHttpResponse(HttpRequest request) {
      this.request = request;
    }

    @Override
    public int statusCode() {
      return 200;
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
      return HttpHeaders.of(Map.of(), (a, b) -> true);
    }

    @Override
    public String body() {
      return "";
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
