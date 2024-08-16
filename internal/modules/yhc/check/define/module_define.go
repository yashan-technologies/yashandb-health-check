package define

const (
	// level 1 modules
	MODULE_OVERVIEW ModuleName = "overview"
	MODULE_HOST     ModuleName = "host_check"
	MODULE_YASDB    ModuleName = "yasdb_check"
	MODULE_OBJECT   ModuleName = "object_check"
	MODULE_SECURITY ModuleName = "security_check"
	MODULE_LOG      ModuleName = "log_analysis"
	MODULE_CUSTOM   ModuleName = "custom_check"

	// the followings are level 2 modules
	// parent module: MN_OVERVIEW
	MODULE_OVERVIEW_HOST  ModuleName = "overview_host"
	MODULE_OVERVIEW_YASDB ModuleName = "overview_yasdb"

	// parent module: MN_HOST
	MODULE_HOST_WORKLOAD ModuleName = "host_workload_check"

	// parent module: MN_YASDB
	MODULE_YASDB_STANDBY     ModuleName = "yasdb_standby_check"
	MODULE_YASDB_CONFIG      ModuleName = "yasdb_config_check"
	MODULE_YASDB_TABLESPACE  ModuleName = "yasdb_tablespace_check"
	MODULE_YASDB_CONTROLFILE ModuleName = "yasdb_controlfile_check"
	MODULE_YASDB_BACKUP      ModuleName = "yasdb_backup_check"
	MODULE_YASDB_WORKLOAD    ModuleName = "yasdb_workload_check"
	MODULE_YASDB_ARCHIVE_LOG ModuleName = "yasdb_archive_log"
	MODULE_YASDB_PERFORMANCE ModuleName = "yasdb_performance_analysis"

	// parent module: MN_OBJECT
	MODULE_OBJECT_NUMBER     ModuleName = "object_number_count"
	MODULE_OBJECT_STATUS     ModuleName = "object_status_check"
	MODULE_OBJECT_TABLE      ModuleName = "object_table_check"
	MODULE_OBJECT_CONSTRAINT ModuleName = "object_constraint_check"
	MODULE_OBJECT_INDEX      ModuleName = "object_index_check"
	MODULE_OBJECT_SEQUENCE   ModuleName = "object_sequence_check"
	MODULE_OBJECT_TASK       ModuleName = "object_task_check"
	MODULE_OBJECT_PACKAGE    ModuleName = "object_package_check"

	// parent module: MN_SECURITY
	MODULE_SECURITY_LOGIN      ModuleName = "security_login_config"
	MODULE_SECURITY_PERMISSION ModuleName = "security_permission_check"
	MODULE_SECURITY_AUDIT      ModuleName = "security_audit_check"

	// parent module: MN_LOG
	MODULE_LOG_RUN   ModuleName = "log_run_analysis"
	MODULE_LOG_REDO  ModuleName = "log_redo_analysis"
	MODULE_LOG_UNDO  ModuleName = "log_undo_analysis"
	MODULE_LOG_ERROR ModuleName = "log_error_analysis"

	MODULE_CUSTOM_BASH ModuleName = "custom_check_bash"
	MODULE_CUSTOM_SQL  ModuleName = "custom_check_sql"
)

type ModuleName string

const (
	METRIC_YASDB_INSTANCE                                                               MetricName = "yasdb_instance"
	METRIC_YASDB_DATABASE                                                               MetricName = "yasdb_database"
	METRIC_YASDB_DEPLOYMENT_ARCHITECTURE                                                MetricName = "yasdb_deployment_architecture"
	METRIC_YASDB_ARCHIVE_THRESHOLD                                                      MetricName = "yasdb_archive_threshold"
	METRIC_YASDB_FILE_PERMISSION                                                        MetricName = "yasdb_file_permission"
	METRIC_YASDB_LISTEN_ADDR                                                            MetricName = "yasdb_listen_address"
	METRIC_YASDB_OS_AUTH                                                                MetricName = "yasdb_os_auth"
	METRIC_HOST_INFO                                                                    MetricName = "host_info"
	METRIC_HOST_FIREWALLD                                                               MetricName = "host_firewalld"
	METRIC_HOST_IPTABLES                                                                MetricName = "host_iptables"
	METRIC_HOST_CPU_INFO                                                                MetricName = "host_cpu_info"
	METRIC_HOST_DISK_INFO                                                               MetricName = "host_disk_info"
	METRIC_HOST_DISK_BLOCK_INFO                                                         MetricName = "host_disk_block_info"
	METRIC_HOST_BIOS_INFO                                                               MetricName = "host_bios_info"
	METRIC_HOST_MEMORY_INFO                                                             MetricName = "host_memory_info"
	METRIC_HOST_NETWORK_INFO                                                            MetricName = "host_network_info"
	METRIC_HOST_HISTORY_CPU_USAGE                                                       MetricName = "host_history_cpu_usage"
	METRIC_HOST_CURRENT_CPU_USAGE                                                       MetricName = "host_current_cpu_usage"
	METRIC_HOST_HISTORY_DISK_IO                                                         MetricName = "host_history_disk_io"
	METRIC_HOST_CURRENT_DISK_IO                                                         MetricName = "host_current_disk_io"
	METRIC_HOST_HISTORY_MEMORY_USAGE                                                    MetricName = "host_history_memory_usage"
	METRIC_HOST_CURRENT_MEMORY_USAGE                                                    MetricName = "host_current_memory_usage"
	METRIC_HOST_HISTORY_NETWORK_IO                                                      MetricName = "host_history_network_io"
	METRIC_HOST_CURRENT_NETWORK_IO                                                      MetricName = "host_current_network_io"
	METRIC_YASDB_ARCHIVE_DEST_STATUS                                                    MetricName = "yasdb_archive_dest_status"
	METRIC_YASDB_ARCHIVE_LOG                                                            MetricName = "yasdb_archive_log"
	METRIC_YASDB_ARCHIVE_LOG_SPACE                                                      MetricName = "yasdb_archive_log_space"
	METRIC_YASDB_PARAMETER                                                              MetricName = "yasdb_parameter"
	METRIC_YASDB_TABLESPACE                                                             MetricName = "yasdb_tablespace"
	METRIC_YASDB_CONTROLFILE_COUNT                                                      MetricName = "yasdb_controlfile_count"
	METRIC_YASDB_CONTROLFILE                                                            MetricName = "yasdb_controlfile"
	METRIC_YASDB_DATAFILE                                                               MetricName = "yasdb_datafile"
	METRIC_YASDB_SESSION                                                                MetricName = "yasdb_session"
	METRIC_YASDB_WAIT_EVENT                                                             MetricName = "yasdb_wait_event"
	METRIC_YASDB_INDEX_OVERSIZED                                                        MetricName = "yasdb_index_oversized"
	METRIC_YASDB_INDEX_TABLE_INDEX_NOT_TOGETHER                                         MetricName = "yasdb_index_table_index_not_together"
	METRIC_YASDB_SEQUENCE_NO_AVAILABLE                                                  MetricName = "yasdb_sequence_no_available"
	METRIC_YASDB_TASK_RUNNING                                                           MetricName = "yasdb_task_running"
	METRIC_YASDB_PACKAGE_NO_PACKAGE_PACKAGE_BODY                                        MetricName = "yasdb_package_no_package_package_body"
	METRIC_YASDB_SECURITY_LOGIN_PASSWORD_STRENGTH                                       MetricName = "yasdb_security_password_strength"
	METRIC_YASDB_AUDITINT_CHECK                                                         MetricName = "yasdb_auditing_check"
	METRIC_YASDB_SECURITY_LOGIN_MAXIMUM_LOGIN_ATTEMPTS                                  MetricName = "yasdb_security_maximum_login_attempts"
	METRIC_YASDB_SECURITY_USER_NO_OPEN                                                  MetricName = "yasdb_security_user_no_open"
	METRIC_YASDB_SECURITY_USER_WITH_SYSTEM_TABLE_PRIVILEGES                             MetricName = "yasdb_security_user_with_system_table_privileges"
	METRIC_YASDB_SECURITY_USER_WITH_DBA_ROLE                                            MetricName = "yasdb_security_user_with_dba_role"
	METRIC_YASDB_SECURITY_USER_ALL_PRIVILEGE_OR_SYSTEM_PRIVILEGES                       MetricName = "yasdb_security_user_all_privilege_or_system_privileges"
	METRIC_YASDB_SECURITY_USER_USE_SYSTEM_TABLESPACE                                    MetricName = "yasdb_security_user_use_system_tablespace"
	METRIC_YASDB_SECURITY_AUDIT_CLEANUP_TASK                                            MetricName = "yasdb_security_audit_cleanup_task"
	METRIC_YASDB_SECURITY_AUDIT_FILE_SIZE                                               MetricName = "yasdb_security_audit_file_size"
	METRIC_YASDB_RUN_LOG_DATABASE_CHANGES                                               MetricName = "yasdb_database_change"
	METRIC_YASDB_SLOW_LOG_PARAMETER                                                     MetricName = "yasdb_slow_log_parameter"
	METRIC_YASDB_SLOW_LOG                                                               MetricName = "yasdb_slow_log"
	METRIC_YASDB_SLOW_LOG_FILE                                                          MetricName = "yasdb_slow_log_file"
	METRIC_YASDB_UNDO_LOG_SIZE                                                          MetricName = "yasdb_undo_size"
	METRIC_YASDB_UNDO_LOG_TOTAL_BLOCK                                                   MetricName = "yasdb_total_undo_block"
	METRIC_YASDB_UNDO_LOG_RUNNING_TRANSACTIONS                                          MetricName = "yasdb_transactions"
	METRIC_YASDB_ALERT_LOG_ERROR                                                        MetricName = "yasdb_alert_log_error"
	METRIC_HOST_DMESG_LOG_ERROR                                                         MetricName = "host_dmesg_log_error"
	METRIC_HOST_SYSTEM_LOG_ERROR                                                        MetricName = "host_system_log_error"
	METRIC_YASDB_BACKUP_SET                                                             MetricName = "yasdb_backup_set"
	METRIC_YASDB_FULL_BACKUP_SET_COUNT                                                  MetricName = "yasdb_full_backup_set_count"
	METRIC_YASDB_BACKUP_SET_PATH                                                        MetricName = "yasdb_backup_set_path"
	METRIC_YASDB_SHARE_POOL                                                             MetricName = "yasdb_share_pool"
	METRIC_YASDB_VM_SWAP_RATE                                                           MetricName = "yasdb_vm_swap_rate"
	METRIC_YASDB_TOP_SQL_BY_CPU_TIME                                                    MetricName = "yasdb_top_sql_by_cpu_time"
	METRIC_YASDB_TOP_SQL_BY_BUFFER_GETS                                                 MetricName = "yasdb_top_sql_by_buffer_gets"
	METRIC_YASDB_TOP_SQL_BY_DISK_READS                                                  MetricName = "yasdb_top_sql_by_disk_reads"
	METRIC_YASDB_TOP_SQL_BY_PARSE_CALLS                                                 MetricName = "yasdb_top_sql_by_parse_calls"
	METRIC_YASDB_HIGH_FREQUENCY_SQL                                                     MetricName = "yasdb_high_frequency_sql"
	METRIC_YASDB_HISTORY_DB_TIME                                                        MetricName = "yasdb_history_db_time"
	METRIC_YASDB_HISTORY_BUFFER_HIT_RATE                                                MetricName = "yasdb_history_buffer_hit_rate"
	METRIC_HOST_HUGE_PAGE                                                               MetricName = "host_huge_page"
	METRIC_HOST_SWAP_MEMORY                                                             MetricName = "host_swap_memory"
	METRIC_YASDB_BUFFER_HIT_RATE                                                        MetricName = "yasdb_buffer_hit_rate"
	METRIC_YASDB_TABLE_LOCK_WAIT                                                        MetricName = "yasdb_table_lock_wait"
	METRIC_YASDB_ROW_LOCK_WAIT                                                          MetricName = "yasdb_row_lock_wait"
	METRIC_YASDB_LONG_RUNNING_TRANSACTION                                               MetricName = "yasdb_long_running_transaction"
	METRIC_YASDB_OBJECT_COUNT                                                           MetricName = "yasdb_object_count"
	METRIC_YASDB_OBJECT_SUMMARY                                                         MetricName = "yasdb_object_summary"
	METRIC_YASDB_SEGMENTS_COUNT                                                         MetricName = "yasdb_segments_count"
	METRIC_YASDB_SEGMENTS_SUMMARY                                                       MetricName = "yasdb_segments_summary"
	METRIC_YASDB_INVALID_OBJECT                                                         MetricName = "yasdb_invalid_object"
	METRIC_YASDB_INVISIBLE_INDEX                                                        MetricName = "yasdb_invisible_index"
	METRIC_YASDB_DISABLED_CONSTRAINT                                                    MetricName = "yasdb_disabled_constraint"
	METRIC_YASDB_TABLE_WITH_TOO_MUCH_COLUMNS                                            MetricName = "yasdb_table_with_too_much_columns"
	METRIC_YASDB_TABLE_WITH_TOO_MUCH_INDEXES                                            MetricName = "yasdb_table_with_too_much_indexes"
	METRIC_YASDB_PARTITIONED_TABLE_WITHOUT_PARTITIONED_INDEXES                          MetricName = "yasdb_partitioned_table_without_partitioned_indexes"
	METRIC_YASDB_TABLE_WITH_ROW_SIZE_EXCEEDS_BLOCK_SIZE                                 MetricName = "yasdb_table_with_row_size_exceeds_block_size"
	METRIC_YASDB_PARTITIONED_TABLE_WITH_NUMBER_OF_HASH_PARTITIONS_IS_NOT_A_POWER_OF_TWO MetricName = "yasdb_partitioned_table_with_number_of_hash_partitions_is_not_a_power_of_two"
	METRIC_YASDB_FOREIGN_KEYS_WITHOUT_INDEXES                                           MetricName = "yasdb_foreign_keys_without_indexes"
	METRIC_YASDB_FOREIGN_KEYS_WITH_IMPLICIT_DATA_TYPE_CONVERSION                        MetricName = "yasdb_foreign_keys_with_implicit_data_type_conversion"
	METRIC_YASDB_INDEX_BLEVEL                                                           MetricName = "yasdb_index_blevel"
	METRIC_YASDB_INDEX_COLUMN                                                           MetricName = "yasdb_index_column"
	METRIC_YASDB_INDEX_INVISIBLE                                                        MetricName = "yasdb_index_invisible"
	METRIC_YASDB_REDO_LOG                                                               MetricName = "yasdb_redo_log"
	METRIC_YASDB_REDO_LOG_COUNT                                                         MetricName = "yasdb_redo_log_count"
	METRIC_YASDB_RUN_LOG_ERROR                                                          MetricName = "yasdb_run_log_error"
)

type MetricName string
