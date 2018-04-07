gogetdef
========

Another [godef] or maybe [gogetdoc].

## Install

go get -u github.com/JohnWall2016/gogetdef

For emacs

cp <gogetdef src dir>/emacs/gogetdef.el ~/.emacs.d/

Add the line below to ~/.emacs

(require 'gogetdef "~/.emacs.d/gogetdef.el")

Restart emacs and open a go code file

M-x gogetdef-describe

M-x gogetdef-jump

M-x gogetdef-all

[godef]: https://github.com/rogpeppe/godef
[gogetdoc]: https://github.com/zmb3/gogetdoc