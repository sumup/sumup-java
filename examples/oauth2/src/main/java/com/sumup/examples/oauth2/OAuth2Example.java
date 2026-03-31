package com.sumup.examples.oauth2;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.scribejava.core.builder.ServiceBuilder;
import com.github.scribejava.core.builder.api.DefaultApi20;
import com.github.scribejava.core.oauth.AccessTokenRequestParams;
import com.github.scribejava.core.oauth.AuthorizationUrlBuilder;
import com.github.scribejava.core.model.OAuth2AccessToken;
import com.github.scribejava.core.oauth.OAuth20Service;
import com.github.scribejava.core.pkce.PKCE;
import com.sumup.sdk.SumUpClient;
import com.sumup.sdk.core.ApiException;
import com.sumup.sdk.models.Merchant;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.security.SecureRandom;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.StringJoiner;

/**
 * OAuth 2.0 Authorization Code flow with SumUp.
 *
 * <p>This example uses ScribeJava to handle the OAuth2 Authorization Code flow with PKCE. Set
 * {@code CLIENT_ID}, {@code CLIENT_SECRET}, and {@code REDIRECT_URI}, then run
 * {@code ./gradlew :examples:oauth2:run}.
 */
public final class OAuth2Example {
  private static final String STATE_COOKIE_NAME = "oauth_state";
  private static final String PKCE_COOKIE_NAME = "oauth_pkce";
  private static final String SCOPES = "email profile";
  private static final ObjectMapper OBJECT_MAPPER = new ObjectMapper();
  private static final SecureRandom SECURE_RANDOM = new SecureRandom();

  private OAuth2Example() {}

  public static void main(String[] args) throws IOException {
    String clientId = requireEnv("CLIENT_ID");
    String clientSecret = requireEnv("CLIENT_SECRET");
    String redirectUri = requireEnv("REDIRECT_URI");

    URI redirect = URI.create(redirectUri);
    String callbackPath = redirect.getPath();
    if (callbackPath == null || callbackPath.isBlank()) {
      callbackPath = "/callback";
    }

    OAuth20Service oauthService =
        new ServiceBuilder(clientId)
            .apiSecret(clientSecret)
            .defaultScope(SCOPES)
            .callback(redirectUri)
            .build(new SumUpOAuthApi());

    int listenPort = redirect.getPort() == -1 ? 8080 : redirect.getPort();
    HttpServer server = HttpServer.create(new InetSocketAddress(listenPort), 0);

    server.createContext(
        "/login",
        exchange -> {
          if (!"GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            sendText(exchange, 405, "Method Not Allowed");
            return;
          }

          String state = randomUrlSafeString(32);
          AuthorizationUrlBuilder authorizationUrlBuilder =
              oauthService.createAuthorizationUrlBuilder().state(state).initPKCE();
          PKCE pkce = authorizationUrlBuilder.getPkce();

          exchange.getResponseHeaders()
              .add("Set-Cookie", buildCookie(STATE_COOKIE_NAME, state));
          exchange.getResponseHeaders()
              .add("Set-Cookie", buildCookie(PKCE_COOKIE_NAME, pkce.getCodeVerifier()));

          String authorizationUrl = authorizationUrlBuilder.build();

          exchange.getResponseHeaders().add("Location", authorizationUrl);
          exchange.sendResponseHeaders(302, -1);
          exchange.close();
        });

    server.createContext(
        callbackPath,
        exchange -> {
          if (!"GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            sendText(exchange, 405, "Method Not Allowed");
            return;
          }

          try {
            handleCallback(exchange, oauthService);
          } catch (Exception ex) {
            sendText(exchange, 500, "OAuth2 error: " + ex.getMessage());
          }
        });

    server.createContext(
        "/",
        exchange -> {
          String body =
              """
              <html>
                <body>
                  <h1>SumUp OAuth2 Example</h1>
                  <p>This example uses ScribeJava for the OAuth2 Authorization Code flow with PKCE.</p>
                  <p><a href="/login">Start OAuth2 Flow</a></p>
                </body>
              </html>
              """;
          sendHtml(exchange, 200, body);
        });

    server.setExecutor(null);
    server.start();
    System.out.printf("Server is running at %s%n", redirectUri);
  }

  private static void handleCallback(HttpExchange exchange, OAuth20Service oauthService)
      throws Exception {
    Map<String, String> queryParams = parseQuery(exchange.getRequestURI().getRawQuery());
    String expectedState = readCookie(exchange, STATE_COOKIE_NAME);
    String codeVerifier = readCookie(exchange, PKCE_COOKIE_NAME);

    if (expectedState == null || codeVerifier == null) {
      sendText(exchange, 400, "Missing OAuth cookies");
      return;
    }

    String state = queryParams.get("state");
    if (state == null || !state.equals(expectedState)) {
      sendText(exchange, 400, "Invalid OAuth state");
      return;
    }

    String code = queryParams.get("code");
    if (code == null || code.isBlank()) {
      sendText(exchange, 400, "Missing authorization code");
      return;
    }

    String merchantCode = queryParams.get("merchant_code");
    if (merchantCode == null || merchantCode.isBlank()) {
      sendText(exchange, 400, "Missing merchant_code query parameter");
      return;
    }

    OAuth2AccessToken accessToken =
        oauthService.getAccessToken(
            AccessTokenRequestParams.create(code).pkceCodeVerifier(codeVerifier));
    SumUpClient client = new SumUpClient(accessToken.getAccessToken());

    try {
      Merchant merchant = client.merchants().getMerchant(merchantCode);
      String body =
          "<pre>"
              + escapeHtml(
                  OBJECT_MAPPER.writerWithDefaultPrettyPrinter().writeValueAsString(merchant))
              + "</pre>";
      sendHtml(exchange, 200, body);
    } catch (ApiException ex) {
      sendText(exchange, ex.getStatusCode(), "Failed to fetch merchant: " + ex.getResponseBody());
    }
  }

  private static String randomUrlSafeString(int byteCount) {
    byte[] bytes = new byte[byteCount];
    SECURE_RANDOM.nextBytes(bytes);
    return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
  }

  private static Map<String, String> parseQuery(String rawQuery) {
    Map<String, String> params = new HashMap<>();
    if (rawQuery == null || rawQuery.isBlank()) {
      return params;
    }

    for (String pair : rawQuery.split("&")) {
      int index = pair.indexOf('=');
      String key = index >= 0 ? decode(pair.substring(0, index)) : decode(pair);
      String value = index >= 0 ? decode(pair.substring(index + 1)) : "";
      params.put(key, value);
    }
    return params;
  }

  private static String decode(String value) {
    return java.net.URLDecoder.decode(value, StandardCharsets.UTF_8);
  }

  private static String buildCookie(String name, String value) {
    return new StringJoiner("; ")
        .add(name + "=" + value)
        .add("Path=/")
        .add("HttpOnly")
        .add("SameSite=Lax")
        .toString();
  }

  private static String readCookie(HttpExchange exchange, String cookieName) {
    List<String> cookieHeaders = exchange.getRequestHeaders().get("Cookie");
    if (cookieHeaders == null) {
      return null;
    }

    for (String header : cookieHeaders) {
      for (String cookie : header.split(";")) {
        String trimmed = cookie.trim();
        if (trimmed.startsWith(cookieName + "=")) {
          return trimmed.substring(cookieName.length() + 1);
        }
      }
    }
    return null;
  }

  private static void sendText(HttpExchange exchange, int statusCode, String body)
      throws IOException {
    byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
    exchange.getResponseHeaders().set("Content-Type", "text/plain; charset=utf-8");
    exchange.sendResponseHeaders(statusCode, bytes.length);
    try (OutputStream outputStream = exchange.getResponseBody()) {
      outputStream.write(bytes);
    }
  }

  private static void sendHtml(HttpExchange exchange, int statusCode, String body)
      throws IOException {
    byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
    exchange.getResponseHeaders().set("Content-Type", "text/html; charset=utf-8");
    exchange.sendResponseHeaders(statusCode, bytes.length);
    try (OutputStream outputStream = exchange.getResponseBody()) {
      outputStream.write(bytes);
    }
  }

  private static String escapeHtml(String input) {
    return input.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;");
  }

  private static String requireEnv(String name) {
    String value = System.getenv(name);
    if (value == null || value.isBlank()) {
      throw new IllegalStateException(name + " environment variable must be set");
    }
    return value;
  }

  private static final class SumUpOAuthApi extends DefaultApi20 {
    @Override
    public String getAccessTokenEndpoint() {
      return "https://api.sumup.com/token";
    }

    @Override
    protected String getAuthorizationBaseUrl() {
      return "https://api.sumup.com/authorize";
    }
  }
}
