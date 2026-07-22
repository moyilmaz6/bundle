#pragma once

#include <filesystem>
#include <functional>
#include <string>
#include <unordered_set>

namespace bundle::launcher {

using PackagePath = std::filesystem::path;

struct StartResult {
  bool started;
  std::string error;
};

class PackageLauncher {
 public:
  using StartRunner = std::function<StartResult(const PackagePath&)>;
  using ReportFailure = std::function<void(const std::string&)>;

  PackageLauncher(StartRunner start_runner, ReportFailure report_failure);

  void Open(const PackagePath& package_path);
  void RunnerExited(const PackagePath& package_path, int exit_status);

 private:
  StartRunner start_runner_;
  ReportFailure report_failure_;
  std::unordered_set<std::string> active_package_paths_;
};

PackagePath NormalizePackagePath(const PackagePath& package_path);

}
