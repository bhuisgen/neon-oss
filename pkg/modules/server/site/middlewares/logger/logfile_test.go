// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"os"
	"testing"
)

func TestCreateLogFileWriter(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				name: os.DevNull,
			},
		},
		{
			name: "error failed to reopen",
			args: args{
				name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateLogFileWriter(tt.args.name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLogFileWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLogFileWriterClose(t *testing.T) {
	f, err := os.Create(os.DevNull)
	if err != nil {
		t.Error("failed to create file")
	}
	defer f.Close()

	type fields struct {
		name string
		flag int
		perm os.FileMode
		f    *os.File
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				name: os.DevNull,
				flag: os.O_WRONLY | os.O_CREATE | os.O_APPEND,
				perm: 0666,
				f:    f,
			},
		},
		{
			name: "error file already close",
			fields: fields{
				name: os.DevNull,
				flag: os.O_WRONLY | os.O_CREATE | os.O_APPEND,
				perm: 0666,
				f:    f,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &logFileWriter{
				name: tt.fields.name,
				flag: tt.fields.flag,
				perm: tt.fields.perm,
				f:    tt.fields.f,
			}
			if err := w.Close(); (err != nil) != tt.wantErr {
				t.Errorf("logFileWriter.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogFileWriterReopen(t *testing.T) {
	f, err := os.Create(os.DevNull)
	if err != nil {
		t.Error("failed to create file")
	}
	defer f.Close()

	type fields struct {
		name string
		flag int
		perm os.FileMode
		f    *os.File
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				name: os.DevNull,
				flag: os.O_WRONLY | os.O_CREATE | os.O_APPEND,
				perm: 0666,
				f:    f,
			},
		},
		{
			name: "error reopen file",
			fields: fields{
				name: "",
				flag: os.O_WRONLY | os.O_CREATE | os.O_APPEND,
				perm: 0666,
				f:    f,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &logFileWriter{
				name: tt.fields.name,
				flag: tt.fields.flag,
				perm: tt.fields.perm,
				f:    tt.fields.f,
			}
			if err := w.Reopen(); (err != nil) != tt.wantErr {
				t.Errorf("logFileWriter.Reopen() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogFileWriterWrite(t *testing.T) {
	f, err := os.Create(os.DevNull)
	if err != nil {
		t.Error("failed to create file")
	}
	defer f.Close()

	type fields struct {
		name string
		flag int
		perm os.FileMode
		f    *os.File
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				name: os.DevNull,
				flag: os.O_WRONLY | os.O_CREATE | os.O_APPEND,
				perm: 0666,
				f:    f,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &logFileWriter{
				name: tt.fields.name,
				flag: tt.fields.flag,
				perm: tt.fields.perm,
				f:    tt.fields.f,
			}
			_, err := w.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("logFileWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLogFileWriterSync(t *testing.T) {
	f, err := os.Create(os.DevNull)
	if err != nil {
		t.Error("failed to create file")
	}
	defer f.Close()

	type fields struct {
		name string
		flag int
		perm os.FileMode
		f    *os.File
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				f: f,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &logFileWriter{
				name: tt.fields.name,
				flag: tt.fields.flag,
				perm: tt.fields.perm,
				f:    tt.fields.f,
			}
			w.Sync()
		})
	}
}
