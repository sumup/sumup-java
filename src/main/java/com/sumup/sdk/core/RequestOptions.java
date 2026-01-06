package com.sumup.sdk.core;

import java.time.Duration;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;

/**
 * Allows per-request overrides for headers, authorization, and timeouts without mutating the shared
 * {@link ApiClient}.
 */
public final class RequestOptions {
  private final String authorizationHeader;
  private final Duration timeout;
  private final Map<String, String> headers;

  private RequestOptions(Builder builder) {
    this.authorizationHeader = builder.authorizationHeader;
    this.timeout = builder.timeout;
    if (builder.headers.isEmpty()) {
      this.headers = Collections.emptyMap();
    } else {
      this.headers = Collections.unmodifiableMap(new LinkedHashMap<>(builder.headers));
    }
  }

  public String authorizationHeader() {
    return authorizationHeader;
  }

  public Duration timeout() {
    return timeout;
  }

  public Map<String, String> headers() {
    return headers;
  }

  public static Builder builder() {
    return new Builder();
  }

  public static final class Builder {
    private String authorizationHeader;
    private Duration timeout;
    private final Map<String, String> headers = new LinkedHashMap<>();

    public Builder authorizationHeader(String authorizationHeader) {
      this.authorizationHeader = authorizationHeader;
      return this;
    }

    public Builder timeout(Duration timeout) {
      this.timeout = timeout;
      return this;
    }

    public Builder header(String name, Object value) {
      Objects.requireNonNull(name, "name");
      Objects.requireNonNull(value, "value");
      headers.put(name, ApiClient.headerValue(value));
      return this;
    }

    public Builder headers(Map<String, ?> values) {
      Objects.requireNonNull(values, "headers");
      values.forEach(
          (name, value) -> {
            Objects.requireNonNull(name, "name");
            Objects.requireNonNull(value, "value");
            headers.put(name, ApiClient.headerValue(value));
          });
      return this;
    }

    public RequestOptions build() {
      return new RequestOptions(this);
    }
  }
}
