package com.sumup.sdk.core;

import com.sumup.sdk.models.Problem;
import java.net.URI;
import java.net.http.HttpHeaders;

/** The API returned an unsuccessful HTTP response. */
public final class ApiStatusException extends ApiException {
  public ApiStatusException(
      String message,
      int status,
      String body,
      Problem problem,
      HttpHeaders headers,
      URI requestUri) {
    super(message, status, body, problem, headers, requestUri);
  }
}
