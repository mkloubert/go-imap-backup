// MIT License
//
// Copyright (c) 2025 Marcel Joachim Kloubert (https://marcel.coffee)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

func compressGZIP(inputPath string, outputPath string) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	// pipe to output path
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, inFile)
	return err
}

func getAllSettings() (AppSettingsList, error) {
	settingList := AppSettingsList{}

	configVars := make([]string, 0)

	envs := os.Environ()
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		key := strings.TrimSpace(parts[0])

		// `IMAP_BACKUP_` with ending digits
		re := regexp.MustCompile(`^(IMAP_BACKUP_)(\d+)$`)
		if !re.MatchString(key) {
			continue
		}

		configName := ""
		if len(parts) > 1 {
			configName = parts[1]
		}

		// save name of env var
		configVars = append(configVars, key)

		// and initialize a new `AppSettings` instance
		settingList[key] = AppSettings{}
		settingList[key][""] = configName // empty key => name
	}

	for _, configVar := range configVars {
		prefix := fmt.Sprintf("%s_", configVar)

		settings := settingList[configVar]

		for _, env := range envs {
			parts := strings.SplitN(env, "=", 2)
			key := strings.TrimSpace(parts[0])

			if !strings.HasPrefix(key, prefix) {
				continue
			}

			configValue := ""
			if len(parts) > 1 {
				configValue = parts[1]
			}

			finalKey := strings.TrimSpace(key[len(prefix):])
			settings[finalKey] = configValue
		}
	}

	return settingList, nil
}

func loadEnvIfExists(cwd string) error {
	envFile := filepath.Join(cwd, ".env")

	if _, err := os.Stat(envFile); err == nil {
		err := godotenv.Load()

		return err
	} else if !os.IsNotExist(err) {
		return err
	}

	return nil
}

func sanitize(s string) string {
	// remove illegal characters
	re := regexp.MustCompile(`[<>:"/\\|?*\r\n]+`)
	s = re.ReplaceAllString(s, "")

	// max 50 chars
	if len(s) > 50 {
		s = s[:50]
	}
	return strings.TrimSpace(s)
}
