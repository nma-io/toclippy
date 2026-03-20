//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

const char* frontmostAppName() {
	NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
	if (app == nil) return "";
	const char *name = [app.localizedName UTF8String];
	return name ? name : "";
}
*/
import "C"

func isForegroundPasswordManager() bool {
	name := C.GoString(C.frontmostAppName())
	return isPasswordManagerName(name)
}
