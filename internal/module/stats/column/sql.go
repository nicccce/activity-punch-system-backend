package column

import "activity-punch-system/internal/global/database"

type briefResult struct {
	Rank              int     `gorm:"ts.rank" json:"rank"`
	TotalScore        float64 `gorm:"ts.total_score" json:"total_score"`
	PunchCount        int64   `gorm:"src.record_count" json:"record_count"`
	TodayPuncherCount int64   `gorm:"tpc.count" json:"today_puncher_count"`
}
type rankResult struct {
	Rank       int     `gorm:"rank" json:"rank"`
	TotalScore float64 `gorm:"total_score" json:"total_score"`
	StudentId  string  `gorm:"student_id" json:"student_id"`
	Name       string  `gorm:"name" json:"name"`
}

const (
	briefSql = `
	  WITH filtered AS (
    SELECT student_id, score, extra_score, created_at
    FROM punch<<<<<<------------------------------------------------------------------------
    WHERE column_id = ? AND created_at <= ? 
	),
	score_summary AS (
    SELECT student_id, SUM(score + extra_score) AS total_score
    FROM filtered
    GROUP BY student_id
	),
	ranked AS (
    SELECT student_id, total_score,
           RANK() OVER (ORDER BY total_score DESC) AS rank
    FROM score_summary
	),
	target_student AS (
    SELECT rank, total_score
    FROM ranked
    WHERE student_id = ?
	),
	student_records_count AS (
    SELECT COUNT(*) AS record_count
    FROM filtered
    WHERE student_id = ?
	),
	today_punch_count AS (
    SELECT COUNT(DISTINCT student_id) AS count
    FROM records
    WHERE column_id = ? AND created_at >= ?
	)
	SELECT
    ts.rank,
    ts.total_score,
    src.record_count,
    tpc.count 
	FROM target_student ts,
     student_records_count src,
     today_punch_count tpc;
	   `
	rankByScoreSql = `
	WITH filtered AS (
    SELECT student_id, score, extra_score
    FROM punch<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<------------------------------
    WHERE column_id = ? AND created_at <= ?
	),
	score_summary AS (
	SELECT 
	student_id, 
	SUM(score + extra_score) AS total_score,
	COUNT(*) AS record_count
    FROM filtered
    GROUP BY student_id
	),

    SELECT 
        ss.student_id, 
        ss.total_score,
        ss.record_count,
        u.name AS user_name,
        RANK() OVER (ORDER BY ss.total_score DESC) AS rank
    FROM score_summary ss
    JOIN user u ON u.id = ss.student_id
	ORDER BY rank
	LIMIT ? OFFSET ?;
`
	rankByCountSql = `
	WITH filtered AS (
    SELECT student_id, score, extra_score
    FROM punch<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<------------------------------
    WHERE column_id = ? AND created_at <= ?
	),
	score_summary AS (
	SELECT 
	student_id, 
	SUM(score + extra_score) AS total_score,
	COUNT(*) AS record_count
    FROM filtered
    GROUP BY student_id
	),

    SELECT 
        ss.student_id, 
        ss.total_score,
        ss.record_count,
        u.name AS user_name,
        RANK() OVER (ORDER BY ss.record_count DESC) AS rank
    FROM score_summary ss
    JOIN user u ON u.id = ss.student_id
	ORDER BY rank
	LIMIT ? OFFSET ?;
`
)

func briefStats(columnId, userId string, askTime int64, result *briefResult) error {
	//这他妈真难写
	return database.DB.Raw(briefSql, columnId, askTime, userId, userId, columnId, askTime-askTime%86400).Scan(result).Error
}

func rankByScore(columnId string, askTime int64, offset, limit int, result *[]rankResult) error {
	return database.DB.Raw(rankByScoreSql, columnId, askTime, limit, offset).Scan(result).Error
}
func rankByCount(columnId string, askTime int64, offset, limit int, result *[]rankResult) error {
	return database.DB.Raw(rankByCountSql, columnId, askTime, limit, offset).Scan(result).Error
}
func selectRecords(columnId string, askTime int64, offset, limit int, result *[]Record, extraOption string, extraParams ...any) error {
	return database.DB.
		Table("punch"). //<<<<<----------------------------------------
		Limit(limit).Offset(offset).Where("column_id = ? AND created_at <= ?", columnId, askTime).
		Where(extraOption, extraParams).
		Order("created_at ASC").
		Find(result).Error
}
func selectRecordsByStudentId(userId string, askTime int64, offset, limit int, result *[]Record) error {
	return database.DB.
		Table("punch"). //<<<<<----------------------------------------
		Limit(limit).Offset(offset).Where("student_id = ? AND created_at <= ?", userId, askTime).
		Order("created_at ASC").
		Find(result).Error
}
