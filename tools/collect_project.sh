#!/bin/bash

# Важно: скрипт рассчитан на запуск из корня проекта: ./tools/collect_project.sh

echo "Изучи код и стрктуру проекта ниже. КОд файлов написан под строкой --FILE."
echo
echo "Как будешь готов - сообщи, мы продолжим проект"

# ---------------------------------------------------------
# 1. Вывод tree -d
# ---------------------------------------------------------
echo "=========================================="
echo "1. PROJECT STRUCTURE (Directories Only)"
echo "=========================================="

if command -v tree &> /dev/null; then
    # Выводим только папки, исключая скрытые и служебные
    tree -d -I '.git|vendor|node_modules|tools'
else
    # Фоллбэк, если tree не установлен
    echo "[INFO] 'tree' command not found, using find instead."
    find . -type d -not -path '*/.*' | sed 's|[^/]*/| |g'
fi

echo "" # Пустая строка для разделения

# ---------------------------------------------------------
# 2. В цикле: путь и содержимое
# ---------------------------------------------------------
echo "=========================================="
echo "2. FILE CONTENTS"
echo "=========================================="

# Ищем файлы. 
# Используем -print0 и read -d '' для безопасной обработки файлов с пробелами в именах
find . -type f \
    -not -path '*/\.*' \
    -not -path '*/vendor/*' \
    -not -path '*/tools/*' \
    -not -name "*.mod" \
    -not -name "*.sum" \
    -not -name "*.exe" \
    -print0 | while IFS= read -r -d '' file; do

    # Проверка на текстовый файл через MIME-тип (самый надежный способ)
    # Если MIME-тип начинается с "text/" (например, text/plain, text/x-go) — выводим
    if file -b --mime-type "$file" | grep -qE '^text/|application/json'; then
        
        # Получаем абсолютный путь для красивой шапки
        FULL_PATH=$(realpath "$file")
        
        echo "--- FILE: $FULL_PATH ---"
        
        # Выводим содержимое
        cat "$file"
        
        # Добавляем пустые строки для читаемости
        echo "" 
        echo "" 
    fi

done
