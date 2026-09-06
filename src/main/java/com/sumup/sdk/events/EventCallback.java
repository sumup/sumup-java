package com.sumup.sdk.events;

/**
 * A synchronous event callback that may throw an application exception.
 *
 * @param <T> notification type
 */
@FunctionalInterface
public interface EventCallback<T extends EventNotification> {
  /**
   * Processes the event. Delivery should be acknowledged only after this returns successfully.
   *
   * @param event verified notification
   * @throws Exception if application processing fails
   */
  void handle(T event) throws Exception;
}
