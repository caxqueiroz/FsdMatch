package com.example.notebook.listener;

import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

@Component
public class NoteEventListener {

    @KafkaListener(topics = "notes-events", groupId = "notebook-indexer")
    public void onNoteEvent(String event) {
        // index into search engine
    }
}
