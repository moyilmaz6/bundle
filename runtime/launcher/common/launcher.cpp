#include "launcher.hpp"

#include <system_error>

namespace bundle::launcher {

PackageLauncher::PackageLauncher(StartRunner start_runner,
                                 ReportFailure report_failure)
    : start_runner_(std::move(start_runner)),
      report_failure_(std::move(report_failure)) {}

void PackageLauncher::Open(const PackagePath& package_path) {
  const auto normalized_path = NormalizePackagePath(package_path);
  const auto path_key = normalized_path.generic_string();
  if (!active_package_paths_.insert(path_key).second) {
    return;
  }

  const auto result = start_runner_(normalized_path);
  if (result.started) {
    return;
  }

  active_package_paths_.erase(path_key);
  if (result.error.empty()) {
    report_failure_("Unable to start " + normalized_path.filename().string() + ".");
    return;
  }
  report_failure_(result.error);
}

void PackageLauncher::RunnerExited(const PackagePath& package_path,
                                   int exit_status) {
  const auto normalized_path = NormalizePackagePath(package_path);
  active_package_paths_.erase(normalized_path.generic_string());
  if (exit_status != 0) {
    report_failure_("Unable to open " + normalized_path.filename().string() + ".");
  }
}

PackagePath NormalizePackagePath(const PackagePath& package_path) {
  std::error_code error;
  const auto canonical_path = std::filesystem::weakly_canonical(package_path, error);
  if (!error) {
    return canonical_path;
  }

  const auto absolute_path = std::filesystem::absolute(package_path, error);
  if (!error) {
    return absolute_path.lexically_normal();
  }
  return package_path.lexically_normal();
}

}
