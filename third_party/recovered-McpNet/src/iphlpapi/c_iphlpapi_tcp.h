#include <windows.h>
#include <iphlpapi.h>
#include <stdio.h>

// 定义 GETEXTENDEDTABLE 函数指针类型
typedef DWORD(WINAPI* GETEXTENDEDTABLE)(PVOID, PDWORD, BOOL, ULONG, TCP_TABLE_CLASS, ULONG);

// 定义 SETTCPENTRY 函数指针类型
typedef DWORD(WINAPI* SETTCPENTRY)(PMIB_TCPROW);

// 定义 GetTcpTable2 函数指针类型
// 注意：系统 iphlpapi.h 已将 GetTcpTable2 声明为函数，此处改名为 PGetTcpTable2
// 以避免“redeclared as different kind of symbol”冲突。结构体 MIB_TCPROW2 /
// MIB_TCPTABLE2 已由系统头文件 tcpmib.h 提供，无需在此重复定义。
typedef DWORD (WINAPI * PGetTcpTable2)(PMIB_TCPTABLE2 TcpTable, PULONG SizePointer, BOOL Order);

// 关闭 TCP 连接初始化
void closeTcpConnectionInit();

// 根据 PID 关闭 TCP 连接
void closeTcpConnectionByPid(DWORD pid, DWORD ulAf);

// 获取指定 TCP 地址和端口的 PID
int getTcpInfoPID(char* Addr, int SunnyProt);
