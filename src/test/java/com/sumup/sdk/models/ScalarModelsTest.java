package com.sumup.sdk.models;

import static org.junit.jupiter.api.Assertions.assertEquals;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;

final class ScalarModelsTest {
  @Test
  void readerScalarsUseJsonStrings() throws Exception {
    var mapper = new ObjectMapper();
    var request = CreateReaderRequest.builder().name("Counter 1").pairingCode("ABC123XYZ").build();
    var json = mapper.readTree(mapper.writeValueAsString(request));
    assertEquals("Counter 1", json.get("name").textValue());
    assertEquals("ABC123XYZ", json.get("pairingCode").textValue());

    var reader =
        mapper.readValue(
            """
        {"id":"reader-123","name":"Counter 1","status":"paired"}
        """,
            Reader.class);
    assertEquals("reader-123", reader.id());
    assertEquals("Counter 1", reader.name());
    assertEquals(ReaderStatus.PAIRED, reader.status());
  }
}
