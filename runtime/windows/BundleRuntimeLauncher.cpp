#include <windows.h>
#include <shellapi.h>

#include <string>

namespace {

std::wstring Quote(const std::wstring& value) {
  return L"\"" + value + L"\"";
}

std::wstring ExecutablePath() {
  wchar_t path[MAX_PATH]{};
  const DWORD length = GetModuleFileNameW(nullptr, path, MAX_PATH);
  return std::wstring(path, length);
}

int RegisterAssociation(const std::wstring& executable) {
  const std::wstring command = Quote(executable) + L" \"%1\"";
  const struct { const wchar_t* key; const wchar_t* value; } entries[] = {
      {L"Software\\Classes\\.bundl", L"com.moyilmaz6.bundl"},
      {L"Software\\Classes\\com.moyilmaz6.bundl", L"Bundle Application"},
      {L"Software\\Classes\\com.moyilmaz6.bundl\\shell\\open\\command", command.c_str()},
  };
  for (const auto& entry : entries) {
    HKEY key{};
    if (RegCreateKeyExW(HKEY_CURRENT_USER, entry.key, 0, nullptr, 0, KEY_SET_VALUE,
                        nullptr, &key, nullptr) != ERROR_SUCCESS) return 1;
    const auto bytes = static_cast<DWORD>((wcslen(entry.value) + 1) * sizeof(wchar_t));
    const auto result = RegSetValueExW(key, nullptr, 0, REG_SZ,
                                       reinterpret_cast<const BYTE*>(entry.value), bytes);
    RegCloseKey(key);
    if (result != ERROR_SUCCESS) return 1;
  }
  SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_IDLIST, nullptr, nullptr);
  return 0;
}

int UnregisterAssociation() {
  HKEY extension_key{};
  wchar_t value[128]{};
  DWORD value_size = sizeof(value);
  if (RegOpenKeyExW(HKEY_CURRENT_USER, L"Software\\Classes\\.bundl", 0,
                    KEY_QUERY_VALUE, &extension_key) == ERROR_SUCCESS) {
    const auto result = RegQueryValueExW(extension_key, nullptr, nullptr, nullptr,
                                         reinterpret_cast<BYTE*>(value), &value_size);
    RegCloseKey(extension_key);
    if (result == ERROR_SUCCESS && std::wstring(value) == L"com.moyilmaz6.bundl") {
      RegDeleteTreeW(HKEY_CURRENT_USER, L"Software\\Classes\\.bundl");
    }
  }
  RegDeleteTreeW(HKEY_CURRENT_USER, L"Software\\Classes\\com.moyilmaz6.bundl");
  SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_IDLIST, nullptr, nullptr);
  return 0;
}

}

int WINAPI wWinMain(HINSTANCE, HINSTANCE, PWSTR, int) {
  int argument_count{};
  LPWSTR* arguments = CommandLineToArgvW(GetCommandLineW(), &argument_count);
  if (arguments == nullptr) return 1;
  const std::wstring executable = ExecutablePath();
  if (argument_count == 2 && std::wstring(arguments[1]) == L"--register") {
    const int result = RegisterAssociation(executable);
    LocalFree(arguments);
    return result;
  }
  if (argument_count == 2 && std::wstring(arguments[1]) == L"--unregister") {
    const int result = UnregisterAssociation();
    LocalFree(arguments);
    return result;
  }
  if (argument_count != 2) {
    MessageBoxW(nullptr, L"Open a .bundl package with Bundle Runtime.", L"Bundle Runtime", MB_OK | MB_ICONWARNING);
    LocalFree(arguments);
    return 2;
  }
  const auto separator = executable.find_last_of(L"\\/");
  const std::wstring runner = executable.substr(0, separator + 1) + L"bundle-runtime-runner.exe";
  std::wstring command = Quote(runner) + L" " + Quote(arguments[1]);
  STARTUPINFOW startup{};
  startup.cb = sizeof(startup);
  PROCESS_INFORMATION process{};
  const BOOL started = CreateProcessW(runner.c_str(), command.data(), nullptr, nullptr, FALSE, 0,
                                      nullptr, nullptr, &startup, &process);
  LocalFree(arguments);
  if (!started) {
    MessageBoxW(nullptr, L"Unable to start the Bundle Runtime runner.", L"Bundle Runtime", MB_OK | MB_ICONERROR);
    return 1;
  }
  CloseHandle(process.hThread);
  CloseHandle(process.hProcess);
  return 0;
}
