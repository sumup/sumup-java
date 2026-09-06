package com.sumup.sdk.events;

/** The event body could not be deserialized. */
public class EventPayloadException extends RuntimeException {
  /** Creates an exception with a description of the failure. */
  public EventPayloadException(String message) {
    super(message);
  }

  /** Creates an exception retaining the original failure. */
  public EventPayloadException(String message, Throwable cause) {
    super(message, cause);
  }
}
