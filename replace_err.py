import os
import glob

def process_file(path):
    with open(path, 'r') as f:
        content = f.read()

    target = """\tif err != nil {
\t\thttp.Error(w, "internal error", http.StatusInternalServerError)
\t\treturn
\t}"""
    replacement = """\tif registry.HandleInternalError(w, err) {
\t\treturn
\t}"""

    if target in content:
        content = content.replace(target, replacement)
        
        # We might need to add the import for "valisgo/internal/registry" if not present
        if '"valisgo/internal/registry"' not in content:
            # simple import addition, looking for "valisgo/internal/domain"
            content = content.replace('"valisgo/internal/domain"', '"valisgo/internal/domain"\n\t"valisgo/internal/registry"')

        with open(path, 'w') as f:
            f.write(content)
        print(f"Updated {path}")

for root, _, files in os.walk('internal/registry'):
    for f in files:
        if f.endswith('.go'):
            process_file(os.path.join(root, f))

