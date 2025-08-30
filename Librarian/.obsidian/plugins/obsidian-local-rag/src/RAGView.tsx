import { ItemView, WorkspaceLeaf } from "obsidian";
import * as React from "react";
import { createRoot, Root } from 'react-dom/client';
import { RAGComponent } from "./RAGComponent";
import LocalRAGPlugin from "../main";

export const RAG_VIEW_TYPE = "rag-view";

export class RAGView extends ItemView {
  private root: Root | null = null;
  private plugin: LocalRAGPlugin;

  constructor(leaf: WorkspaceLeaf, plugin: LocalRAGPlugin) {
    super(leaf);
    this.plugin = plugin;
  }

  getViewType() {
    return RAG_VIEW_TYPE;
  }

  getDisplayText() {
    return "Local RAG";
  }

  async onOpen() {
    this.root = createRoot(this.containerEl.children[1]);
    this.root.render(
      <React.StrictMode>
        {/* Pass both settings and the event emitter as props */}
        <RAGComponent 
            settings={this.plugin.settings} 
            events={this.plugin.events} 
        />
      </React.StrictMode>
    );
  }

  async onClose() {
    this.root?.unmount();
  }
}