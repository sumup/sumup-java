package com.sumup.sdk.events;

import com.fasterxml.jackson.core.type.TypeReference;
import com.sumup.sdk.core.HttpMethod;
import com.sumup.sdk.core.RequestOptions;
import java.net.URI;
import java.net.http.HttpClient;
import java.util.concurrent.CompletableFuture;

/**
 * An event whose affected resource can be fetched using the originating client's configuration.
 *
 * @param <T> resource model
 */
public abstract class FetchableEvent<T> extends EventNotification {
  abstract TypeReference<T> resourceType();

  /** Fetches the resource's current state. Deleted resources may return an API error. */
  public T fetchObject() {
    return fetchObject(null);
  }

  /**
   * Fetches the current resource, rather than its state when the event occurred.
   *
   * @param options optional authentication, headers, and timeout overrides
   * @return current resource, or null for an empty response
   * @throws EventObjectException if the URL has a different origin or the HTTP client follows
   *     redirects
   * @throws com.sumup.sdk.core.ApiException if the API request fails
   */
  public T fetchObject(RequestOptions options) {
    return client().send(HttpMethod.GET, resourcePath(), null, null, null, resourceType(), options);
  }

  /** Fetches the current resource asynchronously. Deleted resources may return an API error. */
  public CompletableFuture<T> fetchObjectAsync() {
    return fetchObjectAsync(null);
  }

  /**
   * Fetches the current resource asynchronously using optional request overrides.
   *
   * @param options optional authentication, headers, and timeout overrides
   * @return future containing the current resource, or null for an empty response
   */
  public CompletableFuture<T> fetchObjectAsync(RequestOptions options) {
    return client()
        .sendAsync(HttpMethod.GET, resourcePath(), null, null, null, resourceType(), options);
  }

  private String resourcePath() {
    if (client().redirectPolicy() != HttpClient.Redirect.NEVER) {
      throw new EventObjectException(
          "Event resource fetching requires an HTTP client with redirects disabled.");
    }
    var resource = URI.create(object().url());
    var baseUri = client().baseUri();
    if (!baseUri.getScheme().equalsIgnoreCase(resource.getScheme())
        || !baseUri.getHost().equalsIgnoreCase(resource.getHost())
        || effectivePort(resource) != effectivePort(baseUri)) {
      throw new EventObjectException(
          "The event resource URL must have the same origin as the API client.");
    }
    var path = resource.getRawPath();
    // A leading // would be interpreted as another authority by URI.resolve.
    if (path.startsWith("//")) {
      throw new EventObjectException("The event resource path must not start with //.");
    }
    return path + (resource.getRawQuery() == null ? "" : "?" + resource.getRawQuery());
  }

  private static int effectivePort(URI uri) {
    return uri.getPort() != -1
        ? uri.getPort()
        : ("https".equalsIgnoreCase(uri.getScheme()) ? 443 : 80);
  }
}
