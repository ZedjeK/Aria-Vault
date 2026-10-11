//go:build windows

package main

import "golang.org/x/sys/windows"

// Page de code UTF-8 de Windows.
const cpUTF8 = 65001

// consoleUTF8 évite les accents illisibles dans l'ancienne console Windows.
func consoleUTF8() { _ = windows.SetConsoleOutputCP(cpUTF8) }
