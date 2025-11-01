from pathlib import Path

def find_project_root(start: Path | None = None,
                      markers=('pyproject.toml', '.git', 'setup.cfg', 'requirements.txt')) -> Path:
    p = (start or Path(__file__).resolve()).absolute()
    for parent in [p] + list(p.parents):
        for m in markers:
            if (parent / m).exists():
                return parent
    # 找不到就退回到当前文件夹
    return p if p.is_dir() else p.parent

