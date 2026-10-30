#!/usr/bin/env python3
# scripts/gen_manifest.py - 根据发布产物生成两份 manifest(版本指针文件)。
#
# 用法:
#   python3 scripts/gen_manifest.py --version 1.2.3 --channel stable \
#     --github-base https://github.com/<owner>/<repo>/releases/download/v1.2.3 \
#     --gitee-base  https://gitee.com/<owner>/<repo>/releases/download/v1.2.3 \
#     --out-dir build/updater \
#     --files "bin/release/skill-box_1.2.3_darwin_universal.zip bin/release/..."
#
# 输出:
#   <out-dir>/manifest.github.json  → assets[].urls 指向 GitHub
#   <out-dir>/manifest.gitee.json   → assets[].urls 指向 Gitee
#
# 两份文件内容除 urls 指向的站点外完全一致,客户端按用户语言选其中一份
# (中文 → Gitee,英文 → GitHub),选哪份安装包就从同一个站下载。
#
# 产物分类规则(按发布命名约定 skill-box_<ver>_<os>_<arch>.<ext>):
#   *.zip                      → 进 manifest,kind=app
#     darwin/universal.zip     → 额外登记 arm64 / amd64 两条 entry(同一个 zip,universal 二进制)
#   *.AppImage                 → 进 manifest,kind=appimage
#   *-installer.exe / *.dmg    → **不进 manifest**(替身脚本装不了),只作为 release 附件供手动下载
#
# Gitee 侧限制:单附件 100MB,超限的 asset 只进 GitHub 那份 manifest。

import argparse
import hashlib
import json
import os
import re
import sys

# 发布命名约定:skill-box_<ver>_<os>_<arch>.<ext>
# arch 允许下划线(x86_64 / aarch64 之类),其余部分用小写字母数字。
NAME_RE = re.compile(
    r"^skill-box_(?P<ver>[0-9][0-9A-Za-z.\-]*)_(?P<os>[a-z0-9]+)_(?P<arch>[a-z0-9_]+)(?P<rest>\..+)$"
)


def sha256_of(path):
    h = hashlib.sha256()
    with open(path, "rb") as fp:
        for chunk in iter(lambda: fp.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def classify(filename):
    """返回 (os, arch, kind, 是否进 manifest, 需要额外登记的 arch 列表)。"""
    # 仅供手动下载的产物:不进 manifest(替身脚本装不了 exe / dmg)
    if filename.endswith("-installer.exe") or filename.lower().endswith(".dmg"):
        return (None, None, "download-only", False, [])

    m = NAME_RE.match(filename)
    if not m:
        return None
    os_name = m.group("os")
    arch = m.group("arch")
    rest = m.group("rest").lower()

    if rest.endswith(".zip"):
        kind = "app"
    elif rest.endswith(".appimage"):
        kind = "appimage"
    else:
        return None

    extra_archs = []
    if os_name == "darwin" and arch == "universal":
        # universal 包同时登记 arm64 / amd64,PickAsset 才能按 GOARCH 命中
        arch = "arm64"
        extra_archs = ["amd64"]
    return (os_name, arch, kind, True, extra_archs)


def build_manifest(args, base_url, skip_oversize):
    assets = []
    for path in args.files:
        path = os.path.normpath(path)
        if not os.path.isfile(path):
            print(f"  skip(不存在): {path}", file=sys.stderr)
            continue
        filename = os.path.basename(path)
        info = classify(filename)
        if info is None:
            print(f"  skip(不符合命名约定): {filename}", file=sys.stderr)
            continue
        os_name, arch, kind, in_manifest, extra_archs = info
        if not in_manifest:
            print(f"  仅手动下载(不进 manifest): {filename}", file=sys.stderr)
            continue
        size = os.path.getsize(path)
        if skip_oversize and size > args.gitee_max_mb * 1024 * 1024:
            print(f"  skip(超过 {args.gitee_max_mb}MB): {filename} ({size // 1024 // 1024} MB)", file=sys.stderr)
            continue
        digest = sha256_of(path)
        url = f"{base_url.rstrip('/')}/{filename}"
        for a in [arch] + extra_archs:
            assets.append({
                "os": os_name,
                "arch": a,
                "kind": kind,
                "size": size,
                "sha256": digest,
                "urls": [url],
            })
    return {
        "channel": args.channel,
        "version": args.version,
        "released_at": args.released_at,
        "min_supported": args.min_supported,
        "notes": {
            "zh-CN": args.note_zh or f"Release {args.version}。",
            "en-US": args.note_en or f"Release {args.version}.",
        },
        "assets": assets,
    }


def main():
    p = argparse.ArgumentParser(description="生成 GitHub / Gitee 两份 manifest 指针文件")
    p.add_argument("--version", required=True)
    p.add_argument("--channel", default="stable")
    p.add_argument("--github-base", required=True, help="GitHub release 下载前缀(含 /download/v<ver>)")
    p.add_argument("--gitee-base", required=True, help="Gitee release 下载前缀(含 /download/v<ver>)")
    p.add_argument("--out-dir", default="build/updater")
    p.add_argument("--files", nargs="+", required=True, help="参与发布的产物路径")
    p.add_argument("--gitee-max-mb", type=int, default=100, help="Gitee 单附件上限,超限只进 GitHub")
    p.add_argument("--min-supported", default="1.0.0")
    p.add_argument("--released-at", default="")
    p.add_argument("--note-zh", default="")
    p.add_argument("--note-en", default="")
    args = p.parse_args()

    os.makedirs(args.out_dir, exist_ok=True)

    targets = [
        (os.path.join(args.out_dir, "manifest.github.json"), args.github_base, False),
        (os.path.join(args.out_dir, "manifest.gitee.json"), args.gitee_base, True),
    ]
    for path, base, skip in targets:
        manifest = build_manifest(args, base, skip)
        with open(path, "w", encoding="utf-8") as fp:
            json.dump(manifest, fp, ensure_ascii=False, indent=2)
            fp.write("\n")
        print(f"  已生成 {path}({len(manifest['assets'])} 个 asset)")


if __name__ == "__main__":
    main()
