#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>

#include <filesystem>
#include <memory>
#include <string>

#include "launcher.hpp"

@interface RuntimeDelegate : NSObject <NSApplicationDelegate> {
 @private
  std::unique_ptr<bundle::launcher::PackageLauncher> launcher_;
  NSMutableDictionary<NSString *, NSTask *> *runners_;
  NSURL *runner_url_;
}

- (void)launchInitialArguments;
@end

@implementation RuntimeDelegate

- (instancetype)init {
  self = [super init];
  if (self == nil) {
    return nil;
  }

  runners_ = [[NSMutableDictionary alloc] init];
  runner_url_ = [[NSBundle mainBundle] URLForResource:@"bundle-runtime-runner"
                                         withExtension:nil];
  launcher_ = std::make_unique<bundle::launcher::PackageLauncher>(
      [self](const bundle::launcher::PackagePath& package_path) {
        return [self startRunnerForPackage:package_path];
      },
      [self](const std::string& message) {
        [self showFailure:message];
      });
  return self;
}

- (void)application:(NSApplication *)application openURLs:(NSArray<NSURL *> *)urls {
  for (NSURL *url in urls) {
    [self openPackageURL:url];
  }
}

- (void)application:(NSApplication *)sender
          openFiles:(NSArray<NSString *> *)filenames {
  for (NSString *filename in filenames) {
    [self openPackageURL:[NSURL fileURLWithPath:filename]];
  }
  [sender replyToOpenOrPrint:NSApplicationDelegateReplySuccess];
}

- (void)launchInitialArguments {
  NSArray<NSString *> *arguments = [[NSProcessInfo processInfo] arguments];
  for (NSUInteger index = 1; index < arguments.count; index++) {
    [self openPackageURL:[NSURL fileURLWithPath:arguments[index]]];
  }
}

- (void)openPackageURL:(NSURL *)package_url {
  if (![package_url isFileURL]) {
    [self showFailure:"Bundle packages must be local files."];
    return;
  }
  launcher_->Open(std::filesystem::path([package_url fileSystemRepresentation]));
}

- (bundle::launcher::StartResult)startRunnerForPackage:
    (const bundle::launcher::PackagePath&)package_path {
  if (runner_url_ == nil) {
    return {false, "The Bundle Runtime runner is missing."};
  }

  const std::string package_string = package_path.string();
  NSString *package_key = [NSString stringWithUTF8String:package_string.c_str()];
  if (package_key == nil) {
    return {false, "Unable to read the Bundle package path."};
  }

  NSTask *task = [[NSTask alloc] init];
  task.executableURL = runner_url_;
  task.arguments = @[ package_key ];
  task.standardOutput = [NSFileHandle fileHandleWithNullDevice];
  task.standardError = [NSFileHandle fileHandleWithNullDevice];
  runners_[package_key] = task;
  task.terminationHandler = ^(NSTask *completed) {
    dispatch_async(dispatch_get_main_queue(), ^{
      [self->runners_ removeObjectForKey:package_key];
      const char *path = [package_key fileSystemRepresentation];
      if (path != nullptr) {
        self->launcher_->RunnerExited(std::filesystem::path(path),
                                      (int)[completed terminationStatus]);
      }
    });
  };

  NSError *error = nil;
  if (![task launchAndReturnError:&error]) {
    [runners_ removeObjectForKey:package_key];
    return {false, "Unable to start " + package_path.filename().string() + "."};
  }
  return {true, ""};
}

- (void)showFailure:(const std::string&)message {
  [NSApp activateIgnoringOtherApps:YES];
  NSAlert *alert = [[NSAlert alloc] init];
  alert.alertStyle = NSAlertStyleWarning;
  alert.messageText = @"Bundle Runtime";
  alert.informativeText = [NSString stringWithUTF8String:message.c_str()];
  [alert runModal];
}

@end

int main(void) {
  @autoreleasepool {
    NSApplication *application = [NSApplication sharedApplication];
    RuntimeDelegate *delegate = [[RuntimeDelegate alloc] init];
    application.delegate = delegate;
    [application setActivationPolicy:NSApplicationActivationPolicyAccessory];
    [delegate launchInitialArguments];
    [application run];
  }
  return 0;
}
