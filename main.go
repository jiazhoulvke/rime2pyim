package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/longbridgeapp/opencc"
)

var output string

func init() {
	flag.StringVar(&output, "output", "", "dict output path")
}

type Block struct {
	Pinpin   string
	Chineses []string
}

func main() {
	flag.Parse()
	dicts := flag.Args()
	t2s, err := opencc.New("t2s")
	if err != nil {
		fmt.Println("打开繁简转换:", err)
		os.Exit(1)
	}
	for _, dict := range dicts {
		ext := strings.ToLower(filepath.Ext(dict))
		if ext == ".pyim" {
			fmt.Println("词典格式无需转换:", dict)
			os.Exit(1)
		}
		if !isExists(dict) {
			fmt.Println("找不到词典:", dict)
			os.Exit(1)
		}
		if err := convertRimeDict2PyimDict(t2s, dict, output); err != nil {
			fmt.Println("转换失败:", err)
			os.Exit(1)
		}
	}
}

var reSpace = regexp.MustCompile(`\s+`)

func convertRimeDict2PyimDict(cc *opencc.OpenCC, dictPath string, savePath string) error {
	basename := filepath.Base(dictPath)
	name := replaceExt(basename, ".pyim")
	var outputPath string
	if savePath == "" {
		outputPath = filepath.Join(filepath.Dir(dictPath), name)
	} else {
		outputPath = filepath.Join(savePath, name)
	}
	dict, err := os.Open(dictPath)
	if err != nil {
		return fmt.Errorf("无法打开词典: %w", err)
	}
	defer dict.Close()
	m := make(map[string]*Block)
	r := bufio.NewReader(dict)
	state := 0
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return fmt.Errorf("%w", err)
			}
			break
		}
		if len(line) == 0 {
			if state == 2 {
				state++
			}
			continue
		}
		if string(line) == "---" {
			state++
		}
		if state == 1 && string(line) == "..." {
			state++
		}
		if state < 2 {
			continue
		}
		matchs := reSpace.FindIndex(line)
		if matchs == nil {
			continue
		}

		chinese := string(line[0:matchs[0]])
		chinese, err = cc.Convert(chinese)
		if err != nil {
			return fmt.Errorf("繁简转换失败: %w", err)
		}
		pinyin := strings.ReplaceAll(string(line[matchs[1]:]), " ", "-")
		b, ok := m[pinyin]
		if !ok {
			b = &Block{
				Pinpin:   pinyin,
				Chineses: []string{chinese},
			}
		} else {
			if inStringSlice(pinyin, b.Chineses) {
				continue
			}
			b.Chineses = append(b.Chineses, chinese)
		}
		m[pinyin] = b
	}
	pinyins := make([]string, 0, len(m))
	for k := range m {
		pinyins = append(pinyins, k)
	}
	sort.Strings(pinyins)
	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("无法创建目标文件: %w", err)
	}
	defer output.Close()
	output.WriteString(";; -*- coding: utf-8 -*--\n")
	for _, pinyin := range pinyins {
		output.WriteString(pinyin)
		b := m[pinyin]
		for _, chinese := range b.Chineses {
			output.Write([]byte(" "))
			output.WriteString(chinese)
		}
		output.Write([]byte("\n"))
	}
	return nil
}

func inStringSlice(s string, slice []string) bool {
	for i := 0; i < len(slice); i++ {
		if s == slice[i] {
			return true
		}
	}
	return false
}

func replaceExt(src string, ext string) string {
	return strings.TrimSuffix(src, filepath.Ext(src)) + ext
}

func isExists(f string) bool {
	_, err := os.Stat(f)
	return !os.IsNotExist(err)
}
