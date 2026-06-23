-- Fix diagnoses stuck at 'under_review' even though officer submitted 'confirmed'/'closed'
UPDATE crop_diagnoses d
SET status = 'reviewed', updated_at = NOW()
WHERE d.status = 'under_review'
  AND EXISTS (
    SELECT 1 FROM diagnosis_reviews r
    WHERE r.diagnosis_id = d.id
      AND r.review_status IN ('confirmed', 'closed')
  );
