Clanker
===

Clanker is an AI agent. Built for fun and learning by hand without AI assistance.


## Design

```mermaid
graph design;
  chatbot-->clanker_gateway;
  clanker_gateway-->llm;
  llm-->clanker_tools;
  clanker_tools-->chat_response;
```