import json
import os

file_path = '/home/jason/code/siyuan/kernel/data/users/users.json'
new_workspace_root = '/mnt/nas-sata12/MindOcean/user-data/notes'

with open(file_path, 'r') as f:
    users = json.load(f)

for user in users:
    if user['username'] == 'jason':
        user['workspace'] = os.path.join(new_workspace_root, 'jason')
        print(f"Updated workspace for jason to {user['workspace']}")

with open(file_path, 'w') as f:
    json.dump(users, f, indent=2)
