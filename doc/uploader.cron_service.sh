#!/bin/sh

SOURCE_DIR="/mnt/usb1_3-1/videos/财学堂录播"
UPLOADING_DIR="/mnt/usb1_3-1/AppUserData/quark/财学堂录播"
DEST="openlist:/quark/videos"
LOG_FILE="/var/log/auto_sync.log"

# 确保两个目录都存在
mkdir -p "$SOURCE_DIR" "$UPLOADING_DIR"

# ========== 阶段 1：自动搬运完成的文件 ==========
for file in "$SOURCE_DIR"/*; do
    # 检查是否真的有文件（防止目录为空时报错）
    [ -f "$file" ] || continue

    # fuser 检测文件是否正在被读写
    if fuser "$file" >/dev/null 2>&1; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')][跳过] 文件仍在写入中: $file" >> "$LOG_FILE"
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [搬运] 文件写入完成，移动到待传区: $file" >> "$LOG_FILE"
        mv "$file" "$UPLOADING_DIR/"
    fi
done

# ========== 阶段 2：执行增量同步 ==========
# 此时 UPLOADING_DIR 里的文件绝对都是完整无损的
# 执行 rclone 同步，并将日志丢弃（或自行修改为输出到日志文件）
rclone move "$UPLOADING_DIR" "$DEST" --size-only -v >> "$LOG_FILE" 2>&1
