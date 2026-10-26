package com.sumup.sdk.core;

import com.sumup.sdk.models.Problem;
import java.net.URI;
import java.net.http.HttpHeaders;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * Exception thrown when the SumUp API responds with a non-success status or the HTTP request fails.
 */
public class ApiException extends RuntimeException {
  private final int statusCode;
  private final String responseBody;
  private final Problem problem;
  private final Map<String, List<String>> responseHeaders;
  private final URI requestUri;

  public ApiException(String message, int statusCode, String responseBody) {
    this(message, statusCode, responseBody, null, null, null);
  }

  public ApiException(
      String message,
      int statusCode,
      String responseBody,
      Problem problem,
      HttpHeaders headers,
      URI requestUri) {
    super(message + " (" + statusCode + ")");
    this.statusCode = statusCode;
    this.responseBody = responseBody;
    this.problem = problem;
    this.responseHeaders = headers == null ? Map.of() : headers.map();
    this.requestUri = requestUri;
  }

  public ApiException(String message, Throwable cause) {
    super(message, cause);
    this.statusCode = -1;
    this.responseBody = null;
    this.problem = null;
    this.responseHeaders = Map.of();
    this.requestUri = null;
  }

  public int getStatusCode() {
    return statusCode;
  }

  public String getResponseBody() {
    return responseBody;
  }

  /** Typed Problem details, when the API returned a Problem response. */
  public Optional<Problem> getProblem() {
    return Optional.ofNullable(problem);
  }

  public Map<String, List<String>> getResponseHeaders() {
    return responseHeaders;
  }

  public Optional<URI> getRequestUri() {
    return Optional.ofNullable(requestUri);
  }
}
