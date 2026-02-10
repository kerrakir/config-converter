## Как работает

Программа конвертирует сетевые конфигурации между Cisco и Huawei, а также в JSON формат.  
Поддерживаются VLAN, access/trunk, L3 Vlanif, маршруты, OSPF, STP, NAT, SMTP и FTP.

---

## Как запускать

В репозитории есть готовый исполняемый файл `converter.exe`.  
Пример запуска для конвертации Cisco → Huawei:

```bash
converter.exe -in examples\cisco_sample.txt -out result.txt -from cisco -to huawei