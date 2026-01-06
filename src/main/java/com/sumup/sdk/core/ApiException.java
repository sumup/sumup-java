package com.sumup.sdk.core;

/**
 * Exception thrown when the SumUp API responds with a non-success status or the HTTP request fails.
 */
public final class ApiException extends RuntimeException {
  private final int statusCode;
  private final String responseBody;

  public ApiException(String message, int statusCode, String responseBody) {
    super(message + " (" + statusCode + ")");
    this.statusCode = statusCode;
    this.responseBody = responseBody;
  }

  public ApiException(String message, Throwable cause) {
    super(message, cause);
    this.statusCode = -1;
    this.responseBody = null;
  }

  public int getStatusCode() {
    return statusCode;
  }

  public String getResponseBody() {
    return responseBody;
  }
}
