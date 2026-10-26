package com.sumup.sdk.core;

/** The request timed out. */
public class ApiTimeoutException extends ApiConnectionException {
  public ApiTimeoutException(String message, Throwable cause) {
    super(message, cause);
  }
}
