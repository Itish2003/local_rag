import { Editor, MarkdownView, Notice, Plugin, WorkspaceLeaf, setIcon } from 'obsidian';
import { RAGView, RAG_VIEW_TYPE } from './src/RAGView';
import { RAGSettingsTab } from './src/RAGSettingsTab';
import { EventEmitter } from './src/EventEmitter';
import { requestUrl } from 'obsidian';

export interface LocalRAGSettings {
	backendUrl: string;
}

const DEFAULT_SETTINGS: LocalRAGSettings = {
	backendUrl: 'http://localhost:8080'
}

export default class LocalRAGPlugin extends Plugin {
	settings: LocalRAGSettings;
	events: EventEmitter = new EventEmitter();
	lastActiveMarkdownView: MarkdownView | null = null;
	
	// --- CORRECTED: Declare the class properties here ---
	statusBarItem: HTMLElement; 
	healthCheckIntervalId: number;

	async onload() {
		await this.loadSettings();
		this.addSettingTab(new RAGSettingsTab(this.app, this));

		// --- Status Bar Setup ---
		this.statusBarItem = this.addStatusBarItem();
		this.updateStatus('loading');
		this.checkBackendStatus();
		this.healthCheckIntervalId = window.setInterval(() => this.checkBackendStatus(), 20000);

		this.registerView(
			RAG_VIEW_TYPE,
			(leaf) => new RAGView(leaf, this)
		);

		this.addRibbonIcon('brain-circuit', 'Open Local RAG', () => {
			this.activateView();
		});

		this.addCommand({
			id: 'query-selection',
			name: 'Query selection with Local RAG',
			editorCallback: (editor: Editor) => {
				const selection = editor.getSelection();
				if (selection) {
					this.activateView();
					this.events.emit('new-query', selection);
				}
			}
		});

		this.registerEvent(
			this.app.workspace.on('editor-menu', (menu, editor, view) => {
				const selection = editor.getSelection();
				if (selection) {
					menu.addItem((item) => {
						item
							.setTitle("Query with Local RAG")
							.setIcon("brain-circuit")
							.onClick(() => {
								this.activateView();
								this.events.emit('new-query', selection);
							});
					});
				}
			})
		);
		
		this.registerEvent(
			this.app.workspace.on('active-leaf-change', (leaf: WorkspaceLeaf | null) => {
				if (leaf && leaf.view instanceof MarkdownView) {
					this.lastActiveMarkdownView = leaf.view;
				}
			})
		);

		this.events.on('insert-text', (text: string) => this.insertTextIntoEditor(text));
		this.events.on('query-start', () => this.updateStatus('querying'));
		this.events.on('query-end', () => this.checkBackendStatus(true));
	}

	async checkBackendStatus(force: boolean = false) {
		// Don't check status if we're in the middle of a query
		if (this.statusBarItem.textContent?.includes("Thinking") && !force) return;
		try {
			await requestUrl({ url: `${this.settings.backendUrl}/health` });
			this.updateStatus('ready');
		} catch (error) {
			this.updateStatus('error');
		}
	}

	updateStatus(status: 'ready' | 'querying' | 'error' | 'loading') {
		if (!this.statusBarItem) return;
		this.statusBarItem.empty();
		
		switch (status) {
			case 'ready':
				setIcon(this.statusBarItem, 'check-circle');
				this.statusBarItem.appendText(' RAG: Ready');
				this.statusBarItem.style.color = 'var(--text-success)';
				break;
			case 'querying':
				setIcon(this.statusBarItem, 'loader');
				this.statusBarItem.appendText(' RAG: Thinking...');
				this.statusBarItem.style.color = 'var(--text-muted)';
				break;
			case 'error':
				setIcon(this.statusBarItem, 'alert-triangle');
				this.statusBarItem.appendText(' RAG: Error');
				this.statusBarItem.style.color = 'var(--text-error)';
				break;
			case 'loading':
				setIcon(this.statusBarItem, 'loader');
				this.statusBarItem.appendText(' RAG: Loading...');
				this.statusBarItem.style.color = 'var(--text-muted)';
				break;
		}
	}
	
	insertTextIntoEditor(text: string) {
		if (this.lastActiveMarkdownView) {
			this.lastActiveMarkdownView.editor.replaceSelection(text);
		} else {
			new Notice("Cannot insert text. Please click into an editor pane first.");
			console.warn("Local RAG: No last active Markdown view to insert text into.");
		}
	}

	async onunload() {
		if (this.healthCheckIntervalId) {
			window.clearInterval(this.healthCheckIntervalId);
		}
	}
	
	async loadSettings() {
		this.settings = Object.assign({}, DEFAULT_SETTINGS, await this.loadData());
	}

	async saveSettings() {
		await this.saveData(this.settings);
		this.checkBackendStatus();
	}

	async activateView() {
		const { workspace } = this.app;
		let existingLeaf = workspace.getLeavesOfType(RAG_VIEW_TYPE)[0];
		if (existingLeaf) {
			workspace.revealLeaf(existingLeaf);
			return;
		}
		const newLeaf = workspace.getRightLeaf(false);
		if (!newLeaf) {
			console.error("Local RAG: Could not create a new leaf.");
			return;
		}
		await newLeaf.setViewState({
			type: RAG_VIEW_TYPE,
			active: true,
		});
		workspace.revealLeaf(newLeaf);
	}
}