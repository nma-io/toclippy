# ToClippy - A cross platform Clipboard Management Tool

### Author: Nicholas Albright (@nma-io)

I was getting tired of trying to use pbpaste/pbcopy on machines that didn't have it, 
so I decided to try my hand at implementing something additional. 

Used Github Copilot to help with the windows api's and some of the autocomplete stuff, 
but the concept was directly from Apple's PBCopy - All credit to them for the amazing
idea. 

Supports both stdin pipe input as well as reading from a file. 

Usage:


> toclippy -i [filename.txt]

or

> cat filename.txt | toclippy

```
Advanced usage of toclippy:
  -clear-all
        securely wipe all clipboard history
  -daemon
        run monitor in background (use with --monitor)
  -f    read clipboard and write to -o file
  -fromcb
        read clipboard and write to -o file
  -history
        browse clipboard history and restore selection
  -i string
        input file
  -maxbuff int
        max buffer size in bytes (default 100MB) (default 104857600)
  -monitor
        monitor clipboard every 5s
  -o string
        output file (use with -fromcb)
  -restore
        browse clipboard history and restore selection
  -update
        check for and install updates
  -utf8
        convert UTF-16 input to UTF-8

```
