-- Postgres check constraint for tenders.equipment.
-- Move 22: TestSQLCheckMatchesGo fails if this list and
-- validEquip in enums.go disagree.
ALTER TABLE tenders
  ADD CONSTRAINT tenders_equipment_check
  CHECK (equipment IN (
    'dry_van', 'flatbed', 'other', 'reefer', 'step_deck'
  ));
