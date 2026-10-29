# primer_count.py counts arm L's primer and Vulcan's guide with one method (parity review R2,
# item 3): python poc/golint/primer_count.py poc/app/golint/PRIMER.md guide.md
#
# For each file: bytes, words (runs of non-space), and pieces, a tokenizer-free proxy for tokens:
# each run of letters, digits, or underscores is one piece, and so is each other character that
# is not a space. Read as UTF-8 with the line ends made LF.

import re, sys

PIECE = re.compile(r'\w+|[^\w\s]')

rows = []
for path in sys.argv[1:]:
    text = open(path, encoding='utf-8').read().replace('\r\n', '\n')
    rows.append((path, len(text.encode('utf-8')), len(text.split()), len(PIECE.findall(text))))
for path, b, w, p in rows:
    print('%-40s %7d bytes %6d words %6d pieces' % (path, b, w, p))
if len(rows) == 2:
    (_, b1, w1, p1), (_, b2, w2, p2) = rows
    print('first / second: bytes %+.1f%%, words %+.1f%%, pieces %+.1f%%' % (100 * (b1 / b2 - 1), 100 * (w1 / w2 - 1), 100 * (p1 / p2 - 1)))
