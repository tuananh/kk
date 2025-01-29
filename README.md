kk
==

Cursor IDE allows you to press Ctrl+K in the terminal and type the question & it will return the command back.

This tool allows you to do the same thing with any terminal.

[![asciicast](https://asciinema.org/a/nlxhlWwdAYrk215h100a651f1.svg)](https://asciinema.org/a/nlxhlWwdAYrk215h100a651f1)

## Usage

If your terminal allows you to bind a key to a command, you can bind `Ctrl+K` to this tool.

If you use Ghostty, you can add the following to your `config`:

```
keybind = ctrl+k=text:kk\x0d
```

If you use Kitty, you can add the following to your `kitty.conf`:

```
map ctrl+k launch --type=overlay kk
```

## License

MIT
