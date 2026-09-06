package com.sumup.sdk.events;

import java.util.concurrent.CompletionStage;

/**
 * An asynchronous callback. Return a stage representing all processing before acknowledgment.
 *
 * @param <T> notification type
 */
@FunctionalInterface
public interface AsyncEventCallback<T extends EventNotification> {
  /**
   * Processes the verified notification.
   *
   * @param event verified notification
   * @return stage completing when processing succeeds
   * @throws Exception if processing fails before returning a stage
   */
  CompletionStage<Void> handle(T event) throws Exception;
}
