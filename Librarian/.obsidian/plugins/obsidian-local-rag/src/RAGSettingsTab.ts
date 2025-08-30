import { App, Notice, PluginSettingTab, Setting } from 'obsidian';
import LocalRAGPlugin from '../main';

export class RAGSettingsTab extends PluginSettingTab {
	plugin: LocalRAGPlugin;

	constructor(app: App, plugin: LocalRAGPlugin) {
		super(app, plugin);
		this.plugin = plugin;
	}

	display(): void {
		const { containerEl } = this;
		containerEl.empty();

		containerEl.createEl('h2', { text: 'Local RAG Settings' });

		// --- Setting 1: Backend URL ---
		new Setting(containerEl)
			.setName('Backend Server URL')
			.setDesc('The address of your Go backend server.')
			.addText(text => text
				.setPlaceholder('http://localhost:8080')
				.setValue(this.plugin.settings.backendUrl)
				.onChange(async (value) => {
					this.plugin.settings.backendUrl = value;
					await this.plugin.saveSettings();
				}));

		// --- NEW: Vault Indexing Helper ---
		containerEl.createEl('h3', { text: 'Vault Indexing' });
		
		// Add descriptive text to guide the user
		const descEl = containerEl.createEl('p');
		descEl.appendText('To allow the backend to index the notes in this vault, you must update its configuration. ');
		descEl.appendText('Copy the path below and paste it into the ');
		descEl.createEl('code', { text: 'INDEX_PATH' });
		descEl.appendText(' variable in your ');
		descEl.createEl('code', { text: 'server/.env' });
		descEl.appendText(' file. You will need to restart the Go server for the change to take effect.');


		// Get the vault path using the Obsidian API
		// @ts-ignore - 'getBasePath' is a public method on the adapter but not in the official API typing yet. It's safe to use.
		const vaultPath = this.app.vault.adapter.getBasePath();

		new Setting(containerEl)
			.setName('Current Vault Path')
			.setDesc('This is the path the backend needs to index your notes.')
			.addText(text => text
				.setValue(vaultPath)
				.setDisabled(true)) // Make the text field read-only
			.addButton(button => button
				.setButtonText("Copy")
				.setCta() // Makes the button more prominent
				.onClick(() => {
					navigator.clipboard.writeText(vaultPath);
					new Notice("Vault path copied to clipboard!");
				}));
	}
}