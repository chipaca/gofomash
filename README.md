# gofomash

`gofomash` started as a little shell script that would call `gofmt` multiple times, with one rule each time (because calling `gofmt` with more than one `-r` will simply replace the earlier rule with the later one), walking a Go subtree but skipping vendor, versioning and build directories.
But then I wanted to give it the ability to express a rule that would expand to an arbitrary number of arguments, and the sed was getting unreadable, so I started rewriting it in Python, stopped, and rewrote it in Go instead.

You can call `gofomash` with multiple rules, and/or with a rule file. It supports any rule that `gofmt` supports, and it additionally supports rules that look like

    f1(f2(a, b…)) -> f3(a, b…)

(note that is a literal U+2026 HORIZONTAL ELLIPSIS) and this will be simply expanded into

    f1(f2(a, b)) -> f3(a, b)
    f1(f2(a, b, c)) -> f3(a, b, c)
    …

you can specify how far out to expand, although it only goes up to 'z'. It does not currently support more than one ellipsised argument but I will fix that at some point. If you need to expand an argument in the _middle_, change the order or use a different alphabet:

    f1(f2(a, c…, b)) -> f3(a, c…, b)

and

    f1(f2(a, b…, 𝛾)) -> f3(a, b…, 𝛾)

will both DTRT.

It does a first pass calling `gofmt -s`, which you can switch to using `gofumpt` instead (with the `-gofumpt` option).
You can also ask tht it does a second pass with `goimports` with the arguments you specify via the `-goimports` option. If you want to call `goimports` with no further arguments, `-goimports=-local` works.

HTH, HAND!
