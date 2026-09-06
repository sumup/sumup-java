package com.sumup.sdk.events;

/** A callback failed. The cause contains the original exception. */
public class EventCallbackException extends RuntimeException {
  /** Creates an exception with a description of the failure. */
  public EventCallbackException(String message) {
    super(message);
  }

  /** Creates an exception retaining the original failure. */
  public EventCallbackException(String message, Throwable cause) {
    super(message, cause);
  }
}
