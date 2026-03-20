use chrono::{Local, NaiveDate, Datelike};

/// Format a date string (YYYY-MM-DD) as a relative human-readable label.
pub fn format_relative(date_str: &str) -> String {
    let Ok(date) = NaiveDate::parse_from_str(date_str, "%Y-%m-%d") else {
        return date_str.to_string();
    };
    let today = Local::now().date_naive();
    let diff = (date - today).num_days();

    match diff {
        0 => "Today".to_string(),
        1 => "Tomorrow".to_string(),
        -1 => "Yesterday".to_string(),
        2..=6 => date.format("%A").to_string(),
        -6..=-2 => format!("Last {}", date.format("%A")),
        _ if date.year() == today.year() => date.format("%b %d").to_string(),
        _ => date.format("%b %d, %Y").to_string(),
    }
}

/// Format a deadline date with severity indication.
/// Returns (label, css_variant) where variant is "normal", "today", or "overdue".
pub fn format_deadline(date_str: &str) -> (String, &'static str) {
    let Ok(date) = NaiveDate::parse_from_str(date_str, "%Y-%m-%d") else {
        return (date_str.to_string(), "normal");
    };
    let today = Local::now().date_naive();
    let diff = (date - today).num_days();

    match diff {
        d if d < 0 => (format!("Overdue · {}", date.format("%b %d")), "overdue"),
        0 => ("Due Today".to_string(), "today"),
        1 => ("Due Tomorrow".to_string(), "normal"),
        2..=6 => (format!("Due {}", date.format("%A")), "normal"),
        _ if date.year() == today.year() => (format!("Due {}", date.format("%b %d")), "normal"),
        _ => (format!("Due {}", date.format("%b %d, %Y")), "normal"),
    }
}

/// Format a section header date for Upcoming/Logbook grouping.
pub fn format_section_date(date_str: &str) -> String {
    let Ok(date) = NaiveDate::parse_from_str(date_str, "%Y-%m-%d") else {
        return date_str.to_string();
    };
    let today = Local::now().date_naive();
    let diff = (date - today).num_days();

    match diff {
        0 => format!("Today — {}", date.format("%a, %b %d")),
        1 => format!("Tomorrow — {}", date.format("%a, %b %d")),
        -1 => format!("Yesterday — {}", date.format("%a, %b %d")),
        2..=6 => date.format("%A, %b %d").to_string(),
        7..=13 => format!("Next Week — {}", date.format("%a, %b %d")),
        _ if date.year() == today.year() => date.format("%b %d").to_string(),
        _ => date.format("%b %d, %Y").to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Local;

    fn today_str() -> String {
        Local::now().date_naive().format("%Y-%m-%d").to_string()
    }

    fn days_from_now(days: i64) -> String {
        let date = Local::now().date_naive() + chrono::Duration::days(days);
        date.format("%Y-%m-%d").to_string()
    }

    #[test]
    fn test_format_today() {
        assert_eq!(format_relative(&today_str()), "Today");
    }

    #[test]
    fn test_format_tomorrow() {
        assert_eq!(format_relative(&days_from_now(1)), "Tomorrow");
    }

    #[test]
    fn test_format_yesterday() {
        assert_eq!(format_relative(&days_from_now(-1)), "Yesterday");
    }

    #[test]
    fn test_format_this_week() {
        let result = format_relative(&days_from_now(3));
        let weekdays = [
            "Monday",
            "Tuesday",
            "Wednesday",
            "Thursday",
            "Friday",
            "Saturday",
            "Sunday",
        ];
        assert!(
            weekdays.iter().any(|w| result == *w),
            "Expected weekday, got: {result}"
        );
    }

    #[test]
    fn test_format_far_future_same_year() {
        let date = Local::now().date_naive() + chrono::Duration::days(60);
        if date.year() == Local::now().date_naive().year() {
            let result = format_relative(&date.format("%Y-%m-%d").to_string());
            assert!(
                result.len() <= 6 || result.contains(' '),
                "Expected 'Mon DD', got: {result}"
            );
        }
    }

    #[test]
    fn test_deadline_overdue() {
        let (label, variant) = format_deadline(&days_from_now(-3));
        assert_eq!(variant, "overdue");
        assert!(
            label.starts_with("Overdue"),
            "Expected 'Overdue...', got: {label}"
        );
    }

    #[test]
    fn test_deadline_today() {
        let (label, variant) = format_deadline(&today_str());
        assert_eq!(label, "Due Today");
        assert_eq!(variant, "today");
    }

    #[test]
    fn test_deadline_tomorrow() {
        let (label, variant) = format_deadline(&days_from_now(1));
        assert_eq!(label, "Due Tomorrow");
        assert_eq!(variant, "normal");
    }

    #[test]
    fn test_invalid_date() {
        assert_eq!(format_relative("not-a-date"), "not-a-date");
    }
}
