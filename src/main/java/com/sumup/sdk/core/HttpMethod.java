package com.sumup.sdk.core;

/** Supported HTTP methods for SumUp API calls. */
public enum HttpMethod {
  GET,
  POST,
  PUT,
  PATCH,
  DELETE,
  OPTIONS,
  HEAD;

  public boolean allowsRequestBody() {
    return switch (this) {
      case POST, PUT, PATCH -> true;
      default -> false;
    };
  }
}
