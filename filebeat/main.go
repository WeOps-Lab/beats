// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/filebeat/cmd"
	inputs "github.com/elastic/beats/v7/filebeat/input/default-inputs"
)

//go:embed module/*
var moduleFiles embed.FS

// The basic model of execution:
// - input: finds files in paths/globs to harvest, starts harvesters
// - harvester: reads a file, sends events to the spooler
// - spooler: buffers events until ready to flush to the publisher
// - publisher: writes to the network, notifies registrar
// - registrar: records positions of files read
// Finally, input uses the registrar information, on restart, to
// determine where in each file to restart a harvester.
func main() {
	// 创建同级的module目录，用于释放module文件
	err := os.MkdirAll("module", 0755)
	if err != nil {
		fmt.Println("无法创建目录:", err)
		os.Exit(1)
	}

	// 释放module文件到module目录
	if err := extractModuleFiles(); err != nil {
		fmt.Println("释放module文件失败:", err)
		os.Exit(1)
	}

	if err := cmd.Filebeat(inputs.Init, cmd.FilebeatSettings()).Execute(); err != nil {
		os.Exit(1)
	}
}

func extractModuleFiles() error {
	return fs.WalkDir(moduleFiles, "module", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel("module", path)
		if err != nil {
			return err
		}

		// 拼接目标路径
		targetPath := filepath.Join("module", relPath)

		if d.IsDir() {
			// 创建目录
			err := os.MkdirAll(targetPath, 0755)
			if err != nil {
				return err
			}
		} else {
			// 读取embed的module文件内容
			file, err := moduleFiles.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			// 创建输出文件
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			// 将embed的module文件内容写入到输出文件
			if _, err := io.Copy(outFile, file); err != nil {
				return err
			}
		}

		return nil
	})
}
