package com.sumup.sdk.events;

/** The resource URL cannot be fetched through this client. */
public class EventObjectException extends RuntimeException {
  /** Creates an exception with a description of the failure. */
  public EventObjectException(String message) {
    super(message);
  }

  /** Creates an exception retaining the original failure. */
  public EventObjectException(String message, Throwable cause) {
    super(message, cause);
  }
}
