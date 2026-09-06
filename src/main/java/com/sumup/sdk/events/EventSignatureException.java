package com.sumup.sdk.events;

/** The signature is missing, malformed, or does not match the original body. */
public class EventSignatureException extends RuntimeException {
  /** Creates an exception with a description of the failure. */
  public EventSignatureException(String message) {
    super(message);
  }

  /** Creates an exception retaining the original failure. */
  public EventSignatureException(String message, Throwable cause) {
    super(message, cause);
  }
}
