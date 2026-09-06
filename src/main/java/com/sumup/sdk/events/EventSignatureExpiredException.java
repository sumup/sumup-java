package com.sumup.sdk.events;

/** The signed delivery timestamp is outside the five-minute acceptance window. */
public class EventSignatureExpiredException extends EventSignatureException {
  /** Creates an exception with a description of the failure. */
  public EventSignatureExpiredException(String message) {
    super(message);
  }

  /** Creates an exception retaining the original failure. */
  public EventSignatureExpiredException(String message, Throwable cause) {
    super(message, cause);
  }
}
