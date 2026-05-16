package com.example.notebook.scheduled;

import com.example.notebook.service.NoteService;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
public class CompactionJob {

    private final NoteService service;

    public CompactionJob(NoteService service) {
        this.service = service;
    }

    @Scheduled(cron = "0 0 2 * * *")
    public void compactSoftDeletedNotes() {
        // walk soft-deleted rows older than 30 days, free storage
    }
}
