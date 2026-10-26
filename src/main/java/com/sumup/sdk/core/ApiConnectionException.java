package com.sumup.sdk.core;

/** The request could not be completed. */
public class ApiConnectionException extends ApiException {
  public ApiConnectionException(String message, Throwable cause) {
    super(message, cause);
  }
}
