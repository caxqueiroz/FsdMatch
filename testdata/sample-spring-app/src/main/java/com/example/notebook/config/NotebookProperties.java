package com.example.notebook.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "notebook")
public class NotebookProperties {

    private int pageSize = 20;
    private int maxPageSize = 100;

    public int getPageSize() { return pageSize; }
    public void setPageSize(int v) { this.pageSize = v; }
    public int getMaxPageSize() { return maxPageSize; }
    public void setMaxPageSize(int v) { this.maxPageSize = v; }
}
