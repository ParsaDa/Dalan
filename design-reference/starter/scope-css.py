#!/usr/bin/env python3
"""Scope the design system's components/bundle.css under .dl and namespace its keyframes.
Usage: scope-css.py <bundle.css> <out.css>   (this produced starter/design.scoped.css)"""
import re, sys

src = re.sub(r'/\*.*?\*/', '', open(sys.argv[1]).read(), flags=re.S)
ROOT = ('.t-dark', '.t-light', '.fx-full', '.fx-light', '.fx-off', '.paused')


def block_end(s, i):
    d = 0
    while i < len(s):
        if s[i] == '{': d += 1
        elif s[i] == '}':
            d -= 1
            if d == 0: return i
        i += 1
    return i


def prefix_sel(sel):
    sel = sel.strip()
    if not sel or sel.startswith('.dl'): return sel
    if sel.startswith(ROOT) or sel.startswith('[dir') or sel.startswith('[lang'): return '.dl' + sel
    return '.dl ' + sel


def split_sel(s):
    parts, d, cur = [], 0, ''
    for ch in s:
        if ch in '([': d += 1
        if ch in ')]': d -= 1
        if ch == ',' and d == 0: parts.append(cur); cur = ''
        else: cur += ch
    parts.append(cur)
    return parts


def process(s):
    res, i = '', 0
    while i < len(s):
        while i < len(s) and s[i].isspace(): i += 1
        if i >= len(s): break
        j = s.find('{', i)
        if j < 0: res += s[i:]; break
        head, e = s[i:j].strip(), block_end(s, j)
        body = s[j + 1:e]
        if head.startswith(('@property', '@keyframes')): res += head + '{' + body + '}\n'
        elif head.startswith('@media'): res += head + '{\n' + process(body) + '}\n'
        else: res += ','.join(prefix_sel(x) for x in split_sel(head)) + '{' + body + '}\n'
        i = e + 1
    return res


css = process(src)
for name in set(re.findall(r'@keyframes ([\w-]+)', css)):
    css = re.sub(r'(?<![\w-])' + re.escape(name) + r'(?![\w-])', 'dl-' + name, css)
open(sys.argv[2], 'w').write(css)
