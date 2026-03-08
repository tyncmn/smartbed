-- Seed: Age/Sex health baseline ranges
-- Source: WHO, AHA, and standard clinical reference ranges
-- Format: age_group, sex, metric_type, min_value, max_value, normal_value, unit
-- ─────────────────────────────────────────────────────────────────────────────

-- ── HEART RATE (bpm) ──────────────────────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'heart_rate', 60, 100, 70, 'bpm'),
('20-29','female', 'heart_rate', 60, 100, 72, 'bpm'),
('30-39','male',   'heart_rate', 60, 100, 71, 'bpm'),
('30-39','female', 'heart_rate', 60, 100, 73, 'bpm'),
('40-49','male',   'heart_rate', 60, 100, 72, 'bpm'),
('40-49','female', 'heart_rate', 62, 103, 75, 'bpm'),
('50-59','male',   'heart_rate', 60, 100, 73, 'bpm'),
('50-59','female', 'heart_rate', 62, 105, 76, 'bpm'),
('60-69','male',   'heart_rate', 60, 100, 74, 'bpm'),
('60-69','female', 'heart_rate', 62, 105, 77, 'bpm'),
('70-79','male',   'heart_rate', 58, 100, 72, 'bpm'),
('70-79','female', 'heart_rate', 60, 105, 75, 'bpm'),
('80+',  'male',   'heart_rate', 55, 100, 70, 'bpm'),
('80+',  'female', 'heart_rate', 58, 105, 73, 'bpm')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── SpO₂ (%) ─────────────────────────────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'spo2', 95, 100, 98, '%'),
('20-29','female', 'spo2', 95, 100, 98, '%'),
('30-39','male',   'spo2', 95, 100, 98, '%'),
('30-39','female', 'spo2', 95, 100, 98, '%'),
('40-49','male',   'spo2', 95, 100, 97, '%'),
('40-49','female', 'spo2', 95, 100, 97, '%'),
('50-59','male',   'spo2', 94, 100, 97, '%'),
('50-59','female', 'spo2', 94, 100, 97, '%'),
('60-69','male',   'spo2', 93, 100, 96, '%'),
('60-69','female', 'spo2', 93, 100, 96, '%'),
('70-79','male',   'spo2', 92, 100, 95, '%'),
('70-79','female', 'spo2', 92, 100, 95, '%'),
('80+',  'male',   'spo2', 90, 100, 94, '%'),
('80+',  'female', 'spo2', 90, 100, 94, '%')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── SKIN TEMPERATURE (°C) ────────────────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'skin_temperature', 34.5, 37.5, 36.6, '°C'),
('20-29','female', 'skin_temperature', 34.5, 37.5, 36.7, '°C'),
('30-39','male',   'skin_temperature', 34.5, 37.5, 36.6, '°C'),
('30-39','female', 'skin_temperature', 34.5, 37.5, 36.7, '°C'),
('40-49','male',   'skin_temperature', 34.4, 37.5, 36.5, '°C'),
('40-49','female', 'skin_temperature', 34.4, 37.5, 36.6, '°C'),
('50-59','male',   'skin_temperature', 34.3, 37.4, 36.4, '°C'),
('50-59','female', 'skin_temperature', 34.3, 37.5, 36.5, '°C'),
('60-69','male',   'skin_temperature', 34.2, 37.3, 36.3, '°C'),
('60-69','female', 'skin_temperature', 34.2, 37.3, 36.4, '°C'),
('70-79','male',   'skin_temperature', 34.0, 37.2, 36.1, '°C'),
('70-79','female', 'skin_temperature', 34.0, 37.2, 36.2, '°C'),
('80+',  'male',   'skin_temperature', 33.8, 37.0, 36.0, '°C'),
('80+',  'female', 'skin_temperature', 33.8, 37.0, 36.0, '°C')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── RESPIRATION RATE (breaths/min) ───────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'respiration', 12, 20, 14, 'br/min'),
('20-29','female', 'respiration', 12, 20, 15, 'br/min'),
('30-39','male',   'respiration', 12, 20, 14, 'br/min'),
('30-39','female', 'respiration', 12, 20, 15, 'br/min'),
('40-49','male',   'respiration', 12, 20, 15, 'br/min'),
('40-49','female', 'respiration', 12, 20, 16, 'br/min'),
('50-59','male',   'respiration', 12, 22, 15, 'br/min'),
('50-59','female', 'respiration', 12, 22, 16, 'br/min'),
('60-69','male',   'respiration', 12, 22, 16, 'br/min'),
('60-69','female', 'respiration', 12, 22, 16, 'br/min'),
('70-79','male',   'respiration', 12, 24, 16, 'br/min'),
('70-79','female', 'respiration', 12, 24, 17, 'br/min'),
('80+',  'male',   'respiration', 12, 25, 17, 'br/min'),
('80+',  'female', 'respiration', 12, 25, 17, 'br/min')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── BLOOD PRESSURE SYSTOLIC (mmHg) ───────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'blood_pressure_systolic', 90, 120, 110, 'mmHg'),
('20-29','female', 'blood_pressure_systolic', 90, 120, 108, 'mmHg'),
('30-39','male',   'blood_pressure_systolic', 90, 125, 114, 'mmHg'),
('30-39','female', 'blood_pressure_systolic', 90, 122, 110, 'mmHg'),
('40-49','male',   'blood_pressure_systolic', 95, 130, 118, 'mmHg'),
('40-49','female', 'blood_pressure_systolic', 92, 128, 114, 'mmHg'),
('50-59','male',   'blood_pressure_systolic', 100, 135, 122, 'mmHg'),
('50-59','female', 'blood_pressure_systolic', 98, 133, 120, 'mmHg'),
('60-69','male',   'blood_pressure_systolic', 100, 140, 126, 'mmHg'),
('60-69','female', 'blood_pressure_systolic', 100, 138, 124, 'mmHg'),
('70-79','male',   'blood_pressure_systolic', 100, 145, 130, 'mmHg'),
('70-79','female', 'blood_pressure_systolic', 100, 143, 128, 'mmHg'),
('80+',  'male',   'blood_pressure_systolic', 100, 150, 132, 'mmHg'),
('80+',  'female', 'blood_pressure_systolic', 100, 148, 130, 'mmHg')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── BLOOD PRESSURE DIASTOLIC (mmHg) ──────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'blood_pressure_diastolic', 60, 80, 70, 'mmHg'),
('20-29','female', 'blood_pressure_diastolic', 60, 80, 69, 'mmHg'),
('30-39','male',   'blood_pressure_diastolic', 60, 80, 72, 'mmHg'),
('30-39','female', 'blood_pressure_diastolic', 60, 80, 70, 'mmHg'),
('40-49','male',   'blood_pressure_diastolic', 62, 85, 75, 'mmHg'),
('40-49','female', 'blood_pressure_diastolic', 62, 83, 73, 'mmHg'),
('50-59','male',   'blood_pressure_diastolic', 65, 85, 77, 'mmHg'),
('50-59','female', 'blood_pressure_diastolic', 65, 83, 75, 'mmHg'),
('60-69','male',   'blood_pressure_diastolic', 65, 90, 78, 'mmHg'),
('60-69','female', 'blood_pressure_diastolic', 65, 88, 76, 'mmHg'),
('70-79','male',   'blood_pressure_diastolic', 65, 90, 78, 'mmHg'),
('70-79','female', 'blood_pressure_diastolic', 65, 88, 76, 'mmHg'),
('80+',  'male',   'blood_pressure_diastolic', 60, 90, 76, 'mmHg'),
('80+',  'female', 'blood_pressure_diastolic', 60, 88, 74, 'mmHg')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── STRESS LEVEL (0-100 scale) ────────────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'stress_level', 0, 40, 20, 'score'),
('20-29','female', 'stress_level', 0, 40, 22, 'score'),
('30-39','male',   'stress_level', 0, 45, 22, 'score'),
('30-39','female', 'stress_level', 0, 45, 25, 'score'),
('40-49','male',   'stress_level', 0, 50, 25, 'score'),
('40-49','female', 'stress_level', 0, 50, 28, 'score'),
('50-59','male',   'stress_level', 0, 45, 23, 'score'),
('50-59','female', 'stress_level', 0, 45, 26, 'score'),
('60-69','male',   'stress_level', 0, 40, 20, 'score'),
('60-69','female', 'stress_level', 0, 40, 22, 'score'),
('70-79','male',   'stress_level', 0, 38, 18, 'score'),
('70-79','female', 'stress_level', 0, 38, 20, 'score'),
('80+',  'male',   'stress_level', 0, 35, 17, 'score'),
('80+',  'female', 'stress_level', 0, 35, 18, 'score')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── SLEEP DURATION (minutes) ──────────────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'sleep_duration', 360, 540, 450, 'min'),
('20-29','female', 'sleep_duration', 360, 540, 450, 'min'),
('30-39','male',   'sleep_duration', 360, 540, 450, 'min'),
('30-39','female', 'sleep_duration', 360, 540, 450, 'min'),
('40-49','male',   'sleep_duration', 360, 510, 435, 'min'),
('40-49','female', 'sleep_duration', 360, 510, 435, 'min'),
('50-59','male',   'sleep_duration', 360, 510, 420, 'min'),
('50-59','female', 'sleep_duration', 360, 510, 420, 'min'),
('60-69','male',   'sleep_duration', 360, 510, 420, 'min'),
('60-69','female', 'sleep_duration', 360, 510, 420, 'min'),
('70-79','male',   'sleep_duration', 330, 480, 405, 'min'),
('70-79','female', 'sleep_duration', 330, 480, 405, 'min'),
('80+',  'male',   'sleep_duration', 300, 480, 390, 'min'),
('80+',  'female', 'sleep_duration', 300, 480, 390, 'min')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;

-- ── MOVEMENT SCORE (0–1 normalized) ───────────────────────────────────────────
INSERT INTO baseline_ranges (age_group, sex, metric_type, min_value, max_value, normal_value, unit) VALUES
('20-29','male',   'movement_score', 0.0, 0.4, 0.15, 'score'),
('20-29','female', 'movement_score', 0.0, 0.4, 0.15, 'score'),
('30-39','male',   'movement_score', 0.0, 0.4, 0.15, 'score'),
('30-39','female', 'movement_score', 0.0, 0.4, 0.15, 'score'),
('40-49','male',   'movement_score', 0.0, 0.5, 0.18, 'score'),
('40-49','female', 'movement_score', 0.0, 0.5, 0.18, 'score'),
('50-59','male',   'movement_score', 0.0, 0.5, 0.20, 'score'),
('50-59','female', 'movement_score', 0.0, 0.5, 0.20, 'score'),
('60-69','male',   'movement_score', 0.0, 0.6, 0.22, 'score'),
('60-69','female', 'movement_score', 0.0, 0.6, 0.22, 'score'),
('70-79','male',   'movement_score', 0.0, 0.6, 0.25, 'score'),
('70-79','female', 'movement_score', 0.0, 0.6, 0.25, 'score'),
('80+',  'male',   'movement_score', 0.0, 0.7, 0.28, 'score'),
('80+',  'female', 'movement_score', 0.0, 0.7, 0.28, 'score')
ON CONFLICT (age_group, sex, metric_type) DO NOTHING;
