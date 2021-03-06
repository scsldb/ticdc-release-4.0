// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/pingcap/parser/model"
	"github.com/pingcap/ticdc/pkg/config"

	"github.com/pingcap/check"
)

func TestSuite(t *testing.T) { check.TestingT(t) }

type decodeFileSuite struct{}

var _ = check.Suite(&decodeFileSuite{})

func (s *decodeFileSuite) TestCanDecodeTOML(c *check.C) {
	dir := c.MkDir()
	path := filepath.Join(dir, "config.toml")
	content := `
case-sensitive = false

[filter]
ignore-txn-start-ts = [1, 2]
ddl-allow-list = [1, 2]
rules = ['*.*', '!test.*']

[mounter]
worker-num = 64

[sink]
dispatchers = [
	{matcher = ['test1.*', 'test2.*'], dispatcher = "ts"},
	{matcher = ['test3.*', 'test4.*'], dispatcher = "rowid"},
]
protocol = "default"

[cyclic-replication]
enable = true
replica-id = 1
filter-replica-ids = [2,3]
id-buckets = 4
sync-ddl = true

[scheduler]
type = "manual"
polling-time = 5
`
	err := ioutil.WriteFile(path, []byte(content), 0644)
	c.Assert(err, check.IsNil)

	cfg := config.GetDefaultReplicaConfig()
	err = strictDecodeFile(path, "cdc", &cfg)
	c.Assert(err, check.IsNil)

	c.Assert(cfg.CaseSensitive, check.IsFalse)
	c.Assert(cfg.Filter, check.DeepEquals, &config.FilterConfig{
		IgnoreTxnStartTs: []uint64{1, 2},
		DDLAllowlist:     []model.ActionType{1, 2},
		Rules:            []string{"*.*", "!test.*"},
	})
	c.Assert(cfg.Mounter, check.DeepEquals, &config.MounterConfig{
		WorkerNum: 64,
	})
	c.Assert(cfg.Sink, check.DeepEquals, &config.SinkConfig{
		DispatchRules: []*config.DispatchRule{
			{Dispatcher: "ts", Matcher: []string{"test1.*", "test2.*"}},
			{Dispatcher: "rowid", Matcher: []string{"test3.*", "test4.*"}},
		},
		Protocol: "default",
	})
	c.Assert(cfg.Cyclic, check.DeepEquals, &config.CyclicConfig{
		Enable:          true,
		ReplicaID:       1,
		FilterReplicaID: []uint64{2, 3},
		IDBuckets:       4,
		SyncDDL:         true,
	})
	c.Assert(cfg.Scheduler, check.DeepEquals, &config.SchedulerConfig{
		Tp:          "manual",
		PollingTime: 5,
	})
}

func (s *decodeFileSuite) TestAndWriteExampleTOML(c *check.C) {
	content := `
# ????????????????????????????????????????????????????????????????????????
# ???????????????????????? filter ??? sink ???????????????????????? true

# Specify whether the schema name and table name in this configuration file are case sensitive
# This configuration will affect both filter and sink related configurations, the default is true
case-sensitive = true

[filter]
# ???????????? StartTs ?????????
# Transactions with the following StartTs will be ignored
ignore-txn-start-ts = [1, 2]

# ???????????????
# ?????????????????????https://docs.pingcap.com/zh/tidb/stable/table-filter#%E8%A1%A8%E5%BA%93%E8%BF%87%E6%BB%A4%E8%AF%AD%E6%B3%95
# The rules of the filter
# Filter rules syntax: https://docs.pingcap.com/tidb/stable/table-filter#syntax
rules = ['*.*', '!test.*']

[mounter]
# mounter ?????????
# the thread number of the the mounter
worker-num = 16

[sink]
# ?????? MQ ?????? Sink??????????????? dispatchers ?????? event ?????????
# ??????????????? default, ts, rowid, table ??????
# For MQ Sinks, you can configure event distribution rules through dispatchers
# Dispatchers support default, ts, rowid and table
dispatchers = [
	{matcher = ['test1.*', 'test2.*'], dispatcher = "ts"},
	{matcher = ['test3.*', 'test4.*'], dispatcher = "rowid"},
]
# ?????? MQ ?????? Sink????????????????????????????????????
# ?????????????????? default, canal ?????????default ??? ticdc-open-protocol
# For MQ Sinks, you can configure the protocol of the messages sending to MQ
# Currently the protocol support default and canal
protocol = "default"

[cyclic-replication]
# ????????????????????????
# Whether to enable cyclic replication
enable = false
# ?????? CDC ????????? ID
# The replica ID of this capture
replica-id = 1
# ???????????????????????? ID
# The replica ID should be ignored
filter-replica-ids = [2,3]
# ???????????? DDL
# Whether to replicate DDL
sync-ddl = true
`
	err := ioutil.WriteFile("changefeed.toml", []byte(content), 0644)
	c.Assert(err, check.IsNil)

	cfg := config.GetDefaultReplicaConfig()
	err = strictDecodeFile("changefeed.toml", "cdc", &cfg)
	c.Assert(err, check.IsNil)

	c.Assert(cfg.CaseSensitive, check.IsTrue)
	c.Assert(cfg.Filter, check.DeepEquals, &config.FilterConfig{
		IgnoreTxnStartTs: []uint64{1, 2},
		Rules:            []string{"*.*", "!test.*"},
	})
	c.Assert(cfg.Mounter, check.DeepEquals, &config.MounterConfig{
		WorkerNum: 16,
	})
	c.Assert(cfg.Sink, check.DeepEquals, &config.SinkConfig{
		DispatchRules: []*config.DispatchRule{
			{Dispatcher: "ts", Matcher: []string{"test1.*", "test2.*"}},
			{Dispatcher: "rowid", Matcher: []string{"test3.*", "test4.*"}},
		},
		Protocol: "default",
	})
	c.Assert(cfg.Cyclic, check.DeepEquals, &config.CyclicConfig{
		Enable:          false,
		ReplicaID:       1,
		FilterReplicaID: []uint64{2, 3},
		SyncDDL:         true,
	})
}

func (s *decodeFileSuite) TestShouldReturnErrForUnknownCfgs(c *check.C) {
	dir := c.MkDir()
	path := filepath.Join(dir, "config.toml")
	content := `filter-case-insensitive = true`
	err := ioutil.WriteFile(path, []byte(content), 0644)
	c.Assert(err, check.IsNil)

	cfg := config.GetDefaultReplicaConfig()
	err = strictDecodeFile(path, "cdc", &cfg)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".*unknown config.*")
}
