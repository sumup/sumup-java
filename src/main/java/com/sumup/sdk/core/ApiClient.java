package com.sumup.sdk.core;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import java.io.IOException;
import java.lang.reflect.Array;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.CompletableFuture;

/** Minimal HTTP client wrapper used by the generated SDK clients. */
public final class ApiClient {
  private static final URI DEFAULT_BASE_URI = URI.create("https://api.sumup.com");
  private static final Duration DEFAULT_REQUEST_TIMEOUT = Duration.ofSeconds(30);
  private static final DateTimeFormatter RFC_1123 = DateTimeFormatter.RFC_1123_DATE_TIME;

  private final HttpClient httpClient;
  private final URI baseUri;
  private final Duration requestTimeout;
  private final ObjectMapper objectMapper;
  private final String accessToken;

  private ApiClient(Builder builder) {
    HttpClient.Builder httpBuilder =
        builder.httpClientBuilder != null ? builder.httpClientBuilder : HttpClient.newBuilder();
    if (builder.connectTimeout != null) {
      httpBuilder.connectTimeout(builder.connectTimeout);
    }
    this.httpClient = builder.httpClient != null ? builder.httpClient : httpBuilder.build();
    this.baseUri = builder.baseUri != null ? builder.baseUri : DEFAULT_BASE_URI;
    this.requestTimeout =
        builder.requestTimeout != null ? builder.requestTimeout : DEFAULT_REQUEST_TIMEOUT;
    this.objectMapper = builder.objectMapper != null ? builder.objectMapper : defaultObjectMapper();
    this.accessToken = builder.accessToken;
  }

  public static Builder builder() {
    return new Builder();
  }

  public <T> T send(
      HttpMethod method,
      String path,
      Map<String, Object> queryParams,
      Map<String, String> headerParams,
      Object body,
      TypeReference<T> responseType) {
    return send(method, path, queryParams, headerParams, body, responseType, null);
  }

  public <T> T send(
      HttpMethod method,
      String path,
      Map<String, Object> queryParams,
      Map<String, String> headerParams,
      Object body,
      TypeReference<T> responseType,
      RequestOptions requestOptions) {
    HttpRequest request =
        buildRequest(method, path, queryParams, headerParams, body, requestOptions);
    try {
      HttpResponse<String> response =
          httpClient.send(request, HttpResponse.BodyHandlers.ofString());
      return readResponse(response, responseType);
    } catch (IOException | InterruptedException e) {
      if (e instanceof InterruptedException) {
        Thread.currentThread().interrupt();
      }
      throw new ApiException("Failed to execute request", e);
    }
  }

  public <T> CompletableFuture<T> sendAsync(
      HttpMethod method,
      String path,
      Map<String, Object> queryParams,
      Map<String, String> headerParams,
      Object body,
      TypeReference<T> responseType) {
    return sendAsync(method, path, queryParams, headerParams, body, responseType, null);
  }

  public <T> CompletableFuture<T> sendAsync(
      HttpMethod method,
      String path,
      Map<String, Object> queryParams,
      Map<String, String> headerParams,
      Object body,
      TypeReference<T> responseType,
      RequestOptions requestOptions) {
    HttpRequest request =
        buildRequest(method, path, queryParams, headerParams, body, requestOptions);
    return httpClient
        .sendAsync(request, HttpResponse.BodyHandlers.ofString())
        .thenApply(
            response -> {
              try {
                return readResponse(response, responseType);
              } catch (IOException e) {
                throw new ApiException("Failed to execute request", e);
              }
            });
  }

  private HttpRequest buildRequest(
      HttpMethod method,
      String path,
      Map<String, Object> queryParams,
      Map<String, String> headerParams,
      Object body,
      RequestOptions requestOptions) {
    Objects.requireNonNull(method, "method");
    Objects.requireNonNull(path, "path");

    HttpRequest.Builder requestBuilder = HttpRequest.newBuilder();
    requestBuilder.uri(resolveUri(path, queryParams));
    requestBuilder.timeout(effectiveTimeout(requestOptions));
    requestBuilder.header("Accept", "application/problem+json, application/json");
    applyAuthorization(requestBuilder, requestOptions);
    applyHeaders(requestBuilder, headerParams, requestOptions);

    boolean hasBody = body != null && method.allowsRequestBody();
    if (hasBody) {
      requestBuilder.header("Content-Type", "application/json");
      requestBuilder.method(method.name(), HttpRequest.BodyPublishers.ofString(writeBody(body)));
    } else {
      requestBuilder.method(method.name(), HttpRequest.BodyPublishers.noBody());
    }
    return requestBuilder.build();
  }

  private <T> T readResponse(HttpResponse<String> response, TypeReference<T> responseType)
      throws IOException {
    int status = response.statusCode();
    if (status / 100 == 2) {
      if (responseType == null) {
        return null;
      }
      String responseBody = response.body();
      if (responseBody == null || responseBody.isEmpty()) {
        return null;
      }
      return objectMapper.readValue(responseBody, responseType);
    }
    throw new ApiException("Request failed", status, response.body());
  }

  private Duration effectiveTimeout(RequestOptions requestOptions) {
    if (requestOptions != null && requestOptions.timeout() != null) {
      return requestOptions.timeout();
    }
    return requestTimeout;
  }

  private void applyAuthorization(
      HttpRequest.Builder requestBuilder, RequestOptions requestOptions) {
    String authorization = requestOptions != null ? requestOptions.authorizationHeader() : null;
    if (authorization == null && accessToken != null && !accessToken.isBlank()) {
      authorization = "Bearer " + accessToken;
    }
    if (authorization != null) {
      requestBuilder.header("Authorization", authorization);
    }
  }

  private void applyHeaders(
      HttpRequest.Builder requestBuilder,
      Map<String, String> headerParams,
      RequestOptions requestOptions) {
    Map<String, String> merged = new LinkedHashMap<>();
    merged.put("User-Agent", SdkMetadata.userAgent());
    merged.putAll(SdkMetadata.runtimeHeaders());
    if (headerParams != null) {
      headerParams.forEach(
          (name, value) -> {
            if (value != null) {
              merged.put(name, value);
            }
          });
    }
    if (requestOptions != null && !requestOptions.headers().isEmpty()) {
      requestOptions
          .headers()
          .forEach(
              (name, value) -> {
                if (value != null) {
                  merged.put(name, value);
                }
              });
    }
    merged.forEach(requestBuilder::header);
  }

  private URI resolveUri(String path, Map<String, Object> queryParams) {
    StringBuilder resolved = new StringBuilder();
    if (path.startsWith("/")) {
      resolved.append(path);
    } else {
      resolved.append('/').append(path);
    }
    String query = buildQueryString(queryParams);
    if (!query.isEmpty()) {
      resolved.append('?').append(query);
    }
    return baseUri.resolve(resolved.toString());
  }

  private String buildQueryString(Map<String, Object> queryParams) {
    if (queryParams == null || queryParams.isEmpty()) {
      return "";
    }
    List<String> parts = new ArrayList<>();
    for (Map.Entry<String, Object> entry : queryParams.entrySet()) {
      appendQueryValue(parts, entry.getKey(), entry.getValue());
    }
    return String.join("&", parts);
  }

  private void appendQueryValue(List<String> parts, String name, Object value) {
    if (value == null) {
      return;
    }
    if (value instanceof Iterable<?> iterable) {
      for (Object element : iterable) {
        appendQueryValue(parts, name, element);
      }
      return;
    }
    if (value.getClass().isArray()) {
      int length = Array.getLength(value);
      for (int i = 0; i < length; i++) {
        appendQueryValue(parts, name, Array.get(value, i));
      }
      return;
    }
    parts.add(urlEncode(name) + "=" + urlEncode(serializeQueryValue(value)));
  }

  private String serializeQueryValue(Object value) {
    if (value instanceof String || value instanceof Number || value instanceof Boolean) {
      return value.toString();
    }
    if (value instanceof OffsetDateTime) {
      return ((OffsetDateTime) value).toString();
    }
    return parameterValue(value);
  }

  private String writeBody(Object body) {
    try {
      return objectMapper.writeValueAsString(body);
    } catch (JsonProcessingException e) {
      throw new ApiException("Failed to serialize request body", e);
    }
  }

  public static String urlEncode(String value) {
    return URLEncoder.encode(value, StandardCharsets.UTF_8);
  }

  public static String parameterValue(Object value) {
    Object unwrapped = unwrapSingleValueRecord(value);
    return unwrapped == null ? null : unwrapped.toString();
  }

  private static Object unwrapSingleValueRecord(Object value) {
    if (value == null || !value.getClass().isRecord()) {
      return value;
    }
    var components = value.getClass().getRecordComponents();
    if (components.length != 1 || !"value".equals(components[0].getName())) {
      return value;
    }
    try {
      return components[0].getAccessor().invoke(value);
    } catch (ReflectiveOperationException e) {
      throw new ApiException("Failed to serialize request parameter", e);
    }
  }

  public static String headerValue(Object value) {
    if (value == null) {
      return null;
    }
    if (value instanceof OffsetDateTime) {
      return RFC_1123.format(((OffsetDateTime) value).withOffsetSameInstant(ZoneOffset.UTC));
    }
    return value.toString();
  }

  private static ObjectMapper defaultObjectMapper() {
    ObjectMapper mapper = new ObjectMapper();
    mapper.registerModule(new JavaTimeModule());
    mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    mapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);
    return mapper;
  }

  public static final class Builder {
    private HttpClient httpClient;
    private HttpClient.Builder httpClientBuilder;
    private Duration connectTimeout;
    private Duration requestTimeout;
    private URI baseUri;
    private ObjectMapper objectMapper;
    private String accessToken;

    public Builder httpClient(HttpClient httpClient) {
      this.httpClient = Objects.requireNonNull(httpClient, "httpClient");
      return this;
    }

    public Builder httpClientBuilder(HttpClient.Builder builder) {
      this.httpClientBuilder = Objects.requireNonNull(builder, "builder");
      return this;
    }

    public Builder baseUri(String baseUri) {
      this.baseUri = URI.create(Objects.requireNonNull(baseUri, "baseUri"));
      return this;
    }

    public Builder baseUri(URI baseUri) {
      this.baseUri = Objects.requireNonNull(baseUri, "baseUri");
      return this;
    }

    public Builder connectTimeout(Duration timeout) {
      this.connectTimeout = timeout;
      return this;
    }

    public Builder requestTimeout(Duration timeout) {
      this.requestTimeout = timeout;
      return this;
    }

    public Builder objectMapper(ObjectMapper objectMapper) {
      this.objectMapper = Objects.requireNonNull(objectMapper, "objectMapper");
      return this;
    }

    public Builder accessToken(String accessToken) {
      this.accessToken = accessToken;
      return this;
    }

    public ApiClient build() {
      return new ApiClient(this);
    }
  }
}
