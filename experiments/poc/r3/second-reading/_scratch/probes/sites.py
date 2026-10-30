# Per-site form: pieces (the method of poc/golint/primer_count.py: a run of word characters is one
# piece, each other non-space character is one piece). V forms for the theories that V does not
# have yet are my own minimal guess at the syntax, and the report says so.
import re

P = re.compile(r'\w+|[^\w\s]')
n = lambda s: len(P.findall(s))

rows = [
    ('T4 requires, L statement', 'requires amt > 0', 'contract.Requires(amt > 0)'),
    ('T4 requires, L comment', 'requires amt > 0', '//@ requires amt > 0'),
    ('T4 ensures, L statement', 'ensures err == nil => result.Amount == amt',
     'defer contract.Ensures(func() bool { return err != nil || r.Amount == amt })'),
    ('T4 ensures, L comment', 'ensures err == nil => result.Amount == amt', '//@ ensures err == nil ==> r.Amount == amt'),
    ('T6 loop invariant, L comment', 'invariant 0 <= i && i <= n', '//@ invariant 0 <= i && i <= n'),
    ('T21 invariant decl', 'invariant Balance >= 0', 'func (a Account) Invariant() bool { return a.Balance >= 0 }'),
    ('T21 port call', 'Checkout.Pay(c, amt)', 's.checkout.Pay(c, amt)'),
    ('T18 port call with ctx', 'Checkout.Pay(c, amt)', 's.checkout.Pay(ctx, c, amt)'),
    ('T7 sum decl, by hand',
     'type Shape enum { Circle{R int}; Square{S int} }',
     'type Shape interface{ isShape() }; type Circle struct{R int}; type Square struct{S int}; func (Circle) isShape() {}; func (Square) isShape() {}'),
    ('T7 sum decl, generator input', 'type Shape enum { Circle{R int}; Square{S int} }', '//vl:sum Shape Circle{R int} Square{S int}'),
    ('T7 match', 'match s { Circle(r) => return r*r; Square(x) => return x*x }',
     'switch v := s.(type) { case Circle: return v.R*v.R; case Square: return v.S*v.S }'),
    ('T7 no-zero mark', 'nozero type Email string', '//vl:nozero\ntype Email string'),
    ('T8 add', 'a + b', 'checked.Add(a, b)'),
    ('T8 map range', 'for k, v := range m {', 'for _, k := range slices.Sorted(maps.Keys(m)) { v := m[k]'),
    ('T8 append', 'ys := append(xs, 1)', 'ys := append(xs[:len(xs):len(xs)], 1)'),
    ('T9 label decl', 'type Email string label personal', '//vl:label personal\ntype Email string'),
    ('T9 derived value', 'lo := strings.ToLower(e)', 'lo := Email(strings.ToLower(string(e)))'),
    ('T9 declassify', 'mail.Send(reveal(e))', 'mail.Send(declassify.Reveal(e))'),
    ('T11 linear mark', 'linear type Conn struct{}', '//vl:linear\ntype Conn struct{}'),
    ('T13 effect in signature', 'func F() !net', '//vl:effects net\nfunc F()'),
    ('T13 poly', 'func Map(xs []int, f func(int) int !e) !e', '//vl:poly f\nfunc Map(xs []int, f func(int) int)'),
    ('T13 effect inside a type', 'fs []func() !net', '//vl:effects fs[] net\nfs []func()'),
    ('T20 send mark', 'send type Counter struct{}', '//vl:send\ntype Counter struct{}'),
    ('T20 iso param', 'func f(b iso *Buf)', '//vl:iso b\nfunc f(b *Buf)'),
    ('T20 iso inside a type', 'c chan iso *Buf', '//vl:iso c<-\nc chan *Buf'),
    ('T10 table row', 'Pending -> Paid, Cancelled', '"Pending": {"Paid", "Cancelled"},'),
    ('T17 mark', 'idempotent handle X(m T) error', '//vl:idempotent\nfunc HandleX(m T) error'),
]
print('%-32s %5s %5s %6s' % ('site', 'V', 'L', 'L/V'))
for name, v, l in rows:
    print('%-32s %5d %5d %6.2f' % (name, n(v), n(l), n(l) / n(v)))

import sys, glob, os
sys.path.insert(0, 'py')
import tiktoken
enc = tiktoken.get_encoding('cl100k_base')
t = lambda s: len(enc.encode(s))
print()
print('same sites, cl100k_base tokens as a proxy')
print('%-32s %5s %5s %6s' % ('site', 'V', 'L', 'L/V'))
for name, v, l in rows:
    print('%-32s %5d %5d %6.2f' % (name, t(v), t(l), t(l) / t(v)))
def arm(root, exts, skip=()):
    total = 0
    for dp, _, fs in os.walk(root):
        for f in fs:
            p = os.path.join(dp, f).replace(os.sep, '/')
            if f.endswith(exts) and not any(s in p for s in skip):
                total += t(open(p, encoding='utf-8').read())
    return total
a = 'src/poc/app/'
v = arm(a + 'vulcan', ('.vulcan',))
l = arm(a + 'golint', ('.go',))
l2 = arm(a + 'golint', ('.go',), ('lib/contract', 'arch/schema', 'arch/config'))
l3 = arm(a + 'golint', ('.go',), ('lib/contract', 'arch/schema', 'arch/config', 'cmd/stockroom'))
print('whole arm with comments: V', v, 'L', l, 'ratio %.3f' % (l / v))
print('L without the fixed files (lib/contract, schema, config):', l2, 'ratio %.3f' % (l2 / v))
print('and with main generated, V without system.vulcan:', l3, v - t(open(a + 'vulcan/system.vulcan', encoding='utf-8').read()), 'ratio %.3f' % (l3 / (v - t(open(a + 'vulcan/system.vulcan', encoding='utf-8').read()))))
print('primer: L', t(open('src/poc/app/golint/PRIMER.md', encoding='utf-8').read()), 'V', t(open('src/docs/guide.md', encoding='utf-8').read()))
for leaf in ['orders', 'billing', 'inventory', 'events', 'purchasing', 'web']:
    lv = sum(t(open(p, encoding='utf-8').read()) for p in glob.glob(a + 'vulcan/api/%s/*' % leaf))
    ll = sum(t(open(p, encoding='utf-8').read()) for p in glob.glob(a + 'golint/api/%s/*' % leaf))
    print('leaf context %-12s V %6d L %6d ratio %.3f' % (leaf, lv, ll, ll / lv))
