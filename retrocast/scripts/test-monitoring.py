from pathlib import Path
import json,unittest
class DashboardContract(unittest.TestCase):
 def test_handler_panel_groups_by_exported_url_label(self):
  dashboard=json.loads((Path(__file__).resolve().parents[1]/'grafana/provisioning/dashboards/retrocast.json').read_text())
  panel=next(p for p in dashboard['panels'] if p['title']=='Request Rate by Handler')
  self.assertIn('by (url)',panel['targets'][0]['expr'])
  self.assertEqual('{{url}}',panel['targets'][0]['legendFormat'])
if __name__=='__main__':unittest.main()
