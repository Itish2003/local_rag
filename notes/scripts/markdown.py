import os

def get_project_structure(project_path, ignore_list=None):
    """
    Recursively walks through a directory and returns a string
    representing the directory structure.
    """
    if ignore_list is None:
        ignore_list = []
    structure = ""
    for root, dirs, files in os.walk(project_path):
        # Filter out ignored directories
        dirs[:] = [d for d in dirs if d not in ignore_list]
        
        level = root.replace(project_path, '').count(os.sep)
        indent = ' ' * 4 * level
        structure += f"{indent}- {os.path.basename(root)}/\n"
        sub_indent = ' ' * 4 * (level + 1)
        for f in files:
            if f not in ignore_list:
                structure += f"{sub_indent}- {f}\n"
    return structure

def create_markdown_from_project(project_path, output_file, ignore_list=None):
    """
    Reads all files in a project and writes their content into a single
    markdown file.
    """
    if ignore_list is None:
        ignore_list = []

    with open(output_file, 'w', encoding='utf-8') as md_file:
        md_file.write(f"# Project: {os.path.basename(project_path)}\n\n")

        # Add the project structure to the markdown file
        md_file.write("## Project Structure\n\n")
        project_structure = get_project_structure(project_path, ignore_list)
        md_file.write(f"```\n{project_structure}\n```\n\n")

        for root, dirs, files in os.walk(project_path):
            # Filter out ignored directories
            dirs[:] = [d for d in dirs if d not in ignore_list]

            for file_name in files:
                if file_name in ignore_list:
                    continue
                file_path = os.path.join(root, file_name)
                relative_path = os.path.relpath(file_path, project_path)
                md_file.write(f"## File: `{relative_path}`\n\n")
                
                try:
                    with open(file_path, 'r', encoding='utf-8', errors='ignore') as file_content:
                        content = file_content.read()
                        file_extension = os.path.splitext(file_name)[1].lstrip('.')
                        md_file.write(f"```{file_extension}\n")
                        md_file.write(content)
                        md_file.write("\n```\n\n")
                except Exception as e:
                    md_file.write(f"Could not read file: {e}\n\n")

if __name__ == '__main__':
    # Replace with the path to your project
    project_directory = './' 
    # The name of the output markdown file
    output_markdown_file = 'project_output.md' 
    
    # Optional: specify files and directories to ignore
    files_and_dirs_to_ignore = ['.git', '__pycache__', '.vscode', 'node_modules','.env', 'my_chroma_data', 'go.mod','package.json','package-lock.json','vite.config.js','eslint.config.js', 'go.sum', output_markdown_file]

    create_markdown_from_project(project_directory, output_markdown_file, files_and_dirs_to_ignore)
    print(f"Project content has been written to {output_markdown_file}")