package com.sumup.sdk.events;

/**
 * Reference to the affected resource; its ID is available without an API request.
 *
 * @param id resource ID
 * @param type resource type, such as member or reader
 * @param url resource API URL
 */
public record EventObjectReference(String id, String type, String url) {}
