package dashboard

import "time"

// DashboardStats represents the main dashboard statistics response
type DashboardStats struct {
	Reports          *ReportStats          `json:"reports"`
	Blogs            *BlogStats            `json:"blogs"`
	PressReleases    *PressReleaseStats    `json:"pressReleases"`
	Users            *UserStats            `json:"users,omitempty"` // Only for admins
	Leads            *LeadStats            `json:"leads"`
	ContentCreation  *ContentCreationStats `json:"contentCreation"`
	Performance      *PerformanceStats     `json:"performance,omitempty"`
}

// ReportStats contains report-related statistics
type ReportStats struct {
	Total         int64        `json:"total"`
	Published     int64        `json:"published"`
	Draft         int64        `json:"draft"`
	Archived      int64        `json:"archived"`
	Change        float64      `json:"change"`        // Percentage change from previous period
	TopPerformers []TopReport  `json:"topPerformers,omitempty"`
}

// BlogStats contains blog-related statistics
type BlogStats struct {
	Total     int64   `json:"total"`
	Published int64   `json:"published"`
	Draft     int64   `json:"draft"`
	Review    int64   `json:"review"`
	Change    float64 `json:"change"`
}

// PressReleaseStats contains press release statistics
type PressReleaseStats struct {
	Total     int64   `json:"total"`
	Published int64   `json:"published"`
	Draft     int64   `json:"draft"`
	Review    int64   `json:"review"`
	Change    float64 `json:"change"`
}

// UserStats contains user-related statistics (admin only)
type UserStats struct {
	Total   int64              `json:"total"`
	Active  int64              `json:"active"`
	ByRole  map[string]int64   `json:"byRole"`
	Change  float64            `json:"change"`
}

// LeadStats contains lead/form submission statistics
type LeadStats struct {
	Total      int64              `json:"total"`
	Pending    int64              `json:"pending"`
	Processed  int64              `json:"processed"`
	ByCategory map[string]int64   `json:"byCategory"`
	Recent     *RecentLeadStats   `json:"recent"`
}

// RecentLeadStats contains recent lead counts
type RecentLeadStats struct {
	Today     int64 `json:"today"`
	ThisWeek  int64 `json:"thisWeek"`
	ThisMonth int64 `json:"thisMonth"`
}

// ContentCreationStats contains content creation trends
type ContentCreationStats struct {
	ReportsThisMonth       int64 `json:"reportsThisMonth"`
	BlogsThisMonth         int64 `json:"blogsThisMonth"`
	PressReleasesThisMonth int64 `json:"pressReleasesThisMonth"`
}

// PerformanceStats contains performance metrics
type PerformanceStats struct {
	TopReports     []TopReport    `json:"topReports,omitempty"`
	TopCategories  []TopCategory  `json:"topCategories,omitempty"`
}

// TopReport represents a top-performing report
type TopReport struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// TopCategory represents a top-performing category
type TopCategory struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	ReportCount int64  `json:"reportCount"`
}

// Activity represents a single activity entry
type Activity struct {
	ID          uint       `json:"id"`
	Type        string     `json:"type"` // ActivityType as string
	Title       string     `json:"title"`
	Description string     `json:"description"`
	EntityType  string     `json:"entityType"`
	EntityID    uint       `json:"entityId"`
	User        *UserInfo  `json:"user"`
	Timestamp   time.Time  `json:"timestamp"`
}

// UserInfo represents minimal user information for activities
type UserInfo struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ActivityResponse represents the activity feed response
type ActivityResponse struct {
	Activities []Activity `json:"activities"`
	Total      int64      `json:"total"`
}

// ActivityType enum for different activity types
type ActivityType string

const (
	ActivityReportPublished       ActivityType = "report_published"
	ActivityReportCreated         ActivityType = "report_created"
	ActivityReportUpdated         ActivityType = "report_updated"
	ActivityReportDeleted         ActivityType = "report_deleted"
	ActivityBlogPublished         ActivityType = "blog_published"
	ActivityBlogCreated           ActivityType = "blog_created"
	ActivityBlogUpdated           ActivityType = "blog_updated"
	ActivityBlogDeleted           ActivityType = "blog_deleted"
	ActivityPressReleasePublished ActivityType = "press_release_published"
	ActivityPressReleaseCreated   ActivityType = "press_release_created"
	ActivityPressReleaseUpdated   ActivityType = "press_release_updated"
	ActivityPressReleaseDeleted   ActivityType = "press_release_deleted"
	ActivityUserCreated           ActivityType = "user_created"
	ActivityUserUpdated           ActivityType = "user_updated"
	ActivityUserDeleted           ActivityType = "user_deleted"
	ActivityUserRoleChanged       ActivityType = "user_role_changed"
	ActivityLeadReceived          ActivityType = "lead_received"
	ActivityLeadProcessed         ActivityType = "lead_processed"
	ActivityAuthLogin             ActivityType = "auth_login"
	ActivityAuthLoginFailed       ActivityType = "auth_login_failed"
	ActivityAuthLogout            ActivityType = "auth_logout"
	ActivityAuthTokenRefresh      ActivityType = "auth_token_refresh"
)

// GetActivityTitle returns a human-readable title for an activity type
func GetActivityTitle(activityType ActivityType) string {
	titles := map[ActivityType]string{
		ActivityReportPublished:       "Report Published",
		ActivityReportCreated:         "Report Created",
		ActivityReportUpdated:         "Report Updated",
		ActivityReportDeleted:         "Report Deleted",
		ActivityBlogPublished:         "Blog Published",
		ActivityBlogCreated:           "Blog Created",
		ActivityBlogUpdated:           "Blog Updated",
		ActivityBlogDeleted:           "Blog Deleted",
		ActivityPressReleasePublished: "Press Release Published",
		ActivityPressReleaseCreated:   "Press Release Created",
		ActivityPressReleaseUpdated:   "Press Release Updated",
		ActivityPressReleaseDeleted:   "Press Release Deleted",
		ActivityUserCreated:           "User Created",
		ActivityUserUpdated:           "User Updated",
		ActivityUserDeleted:           "User Deleted",
		ActivityUserRoleChanged:       "User Role Changed",
		ActivityLeadReceived:          "Lead Received",
		ActivityLeadProcessed:         "Lead Processed",
		ActivityAuthLogin:             "User Logged In",
		ActivityAuthLoginFailed:       "Login Failed",
		ActivityAuthLogout:            "User Logged Out",
		ActivityAuthTokenRefresh:      "Session Refreshed",
	}
	if title, ok := titles[activityType]; ok {
		return title
	}
	return string(activityType)
}
