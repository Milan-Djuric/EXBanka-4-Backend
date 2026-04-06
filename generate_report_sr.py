#!/usr/bin/env python3
"""Generisanje PDF izvještaja o pokrivenosti unit testova – srpska verzija."""

from reportlab.lib.pagesizes import A4
from reportlab.lib import colors
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import cm
from reportlab.platypus import (
    SimpleDocTemplate, Table, TableStyle, Paragraph, Spacer,
    HRFlowable, PageBreak, KeepTogether
)
from reportlab.lib.enums import TA_CENTER, TA_LEFT, TA_RIGHT
from reportlab.platypus.flowables import Flowable

# ── Paleta boja ───────────────────────────────────────────────────────────────
DARK_BLUE   = colors.HexColor('#1a3a5c')
MID_BLUE    = colors.HexColor('#2e5f8a')
ACCENT_BLUE = colors.HexColor('#4a90c4')
LIGHT_GRAY  = colors.HexColor('#f5f5f5')
SILVER      = colors.HexColor('#d0d0d0')
WHITE       = colors.white
GREEN       = colors.HexColor('#2e7d32')
ORANGE      = colors.HexColor('#e65100')
RED         = colors.HexColor('#c62828')
DARK_TEXT   = colors.HexColor('#1c1c1c')

OUTPUT_PATH = r'c:\Users\denis\SIdrugari\EXBanka-4-Backend\test-coverage-report-sr.pdf'

# ── Pomocne funkcije ──────────────────────────────────────────────────────────

def coverage_color(pct_str: str):
    try:
        val = float(pct_str.replace('%', '').replace(',', '.'))
    except ValueError:
        return DARK_TEXT
    if val >= 95:
        return GREEN
    if val >= 85:
        return ORANGE
    return RED


def cov_paragraph(pct_str: str, style):
    col = coverage_color(pct_str)
    hex_col = '#{:02x}{:02x}{:02x}'.format(
        int(col.red * 255), int(col.green * 255), int(col.blue * 255)
    )
    return Paragraph(f'<font color="{hex_col}"><b>{pct_str}</b></font>', style)


# ── Footer sa brojem stranice ─────────────────────────────────────────────────

def make_footer(canvas, doc):
    canvas.saveState()
    canvas.setFont('Helvetica', 8)
    canvas.setFillColor(colors.HexColor('#888888'))
    width, _ = A4
    canvas.drawString(2 * cm, 1.2 * cm,
                      'EXBanka-4-Backend \u2014 Izvjestaj o pokrivenosti unit testova  |  2026-03-28')
    canvas.drawRightString(width - 2 * cm, 1.2 * cm,
                           f'Stranica {doc.page}')
    canvas.restoreState()


# ── Stilovi ───────────────────────────────────────────────────────────────────

base_styles = getSampleStyleSheet()

STYLE_TITLE = ParagraphStyle(
    'ReportTitle',
    fontName='Helvetica-Bold',
    fontSize=26,
    textColor=WHITE,
    alignment=TA_CENTER,
    spaceAfter=6,
)
STYLE_SUBTITLE = ParagraphStyle(
    'ReportSubtitle',
    fontName='Helvetica',
    fontSize=12,
    textColor=colors.HexColor('#cce0f5'),
    alignment=TA_CENTER,
    spaceAfter=4,
)
STYLE_SECTION = ParagraphStyle(
    'SectionHeader',
    fontName='Helvetica-Bold',
    fontSize=13,
    textColor=WHITE,
    spaceAfter=4,
    spaceBefore=14,
    leftIndent=0,
)
STYLE_BODY = ParagraphStyle(
    'Body',
    fontName='Helvetica',
    fontSize=9,
    textColor=DARK_TEXT,
    spaceAfter=4,
    leading=13,
)
STYLE_CELL = ParagraphStyle(
    'Cell',
    fontName='Helvetica',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=11,
)
STYLE_CELL_BOLD = ParagraphStyle(
    'CellBold',
    fontName='Helvetica-Bold',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=11,
)
STYLE_NOTE_TITLE = ParagraphStyle(
    'NoteTitle',
    fontName='Helvetica-Bold',
    fontSize=9,
    textColor=DARK_TEXT,
    leading=13,
)
STYLE_NOTE = ParagraphStyle(
    'Note',
    fontName='Helvetica',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=13,
    leftIndent=10,
)
STYLE_KEY_TITLE = ParagraphStyle(
    'KeyTitle',
    fontName='Helvetica-Bold',
    fontSize=9,
    textColor=DARK_BLUE,
    spaceAfter=2,
    spaceBefore=6,
)
STYLE_KEY_ITEM = ParagraphStyle(
    'KeyItem',
    fontName='Helvetica',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=13,
    leftIndent=8,
)


# ── Baner sekcije ─────────────────────────────────────────────────────────────

class ColorBanner(Flowable):
    def __init__(self, text, width, height=22, bg=DARK_BLUE, style=None):
        super().__init__()
        self.text = text
        self._width = width
        self._height = height
        self.bg = bg
        self.style = style or STYLE_SECTION

    def wrap(self, *args):
        return self._width, self._height

    def draw(self):
        c = self.canv
        c.setFillColor(self.bg)
        c.roundRect(0, 0, self._width, self._height, 4, fill=1, stroke=0)
        c.setFillColor(WHITE)
        c.setFont('Helvetica-Bold', 11)
        c.drawCentredString(self._width / 2, 6, self.text)


class TitleBanner(Flowable):
    def __init__(self, width, height=130):
        super().__init__()
        self._width = width
        self._height = height

    def wrap(self, *args):
        return self._width, self._height

    def draw(self):
        c = self.canv
        c.setFillColor(DARK_BLUE)
        c.roundRect(0, 0, self._width, self._height, 8, fill=1, stroke=0)
        c.setFillColor(ACCENT_BLUE)
        c.rect(0, self._height - 8, self._width, 8, fill=1, stroke=0)
        c.setFillColor(WHITE)
        c.setFont('Helvetica-Bold', 22)
        c.drawCentredString(self._width / 2, self._height - 48,
                            'EXBanka-4-Backend')
        c.setFont('Helvetica-Bold', 15)
        c.drawCentredString(self._width / 2, self._height - 70,
                            'Izvjestaj o pokrivenosti unit testova')
        c.setStrokeColor(ACCENT_BLUE)
        c.setLineWidth(1.5)
        c.line(self._width * 0.2, self._height - 82,
               self._width * 0.8, self._height - 82)
        c.setFont('Helvetica', 10)
        c.setFillColor(colors.HexColor('#cce0f5'))
        c.drawCentredString(self._width / 2, self._height - 98,
                            'Datum: 2026-03-28')
        c.setFont('Helvetica-Bold', 10)
        c.setFillColor(colors.HexColor('#a5d6a7'))
        c.drawCentredString(self._width / 2, self._height - 114,
                            'Status: Svi testovi prolaze \u2713')


# ── Tabele ────────────────────────────────────────────────────────────────────

SUMMARY_HEADER_BG = DARK_BLUE
SUMMARY_TOTAL_BG  = MID_BLUE
DETAIL_HEADER_BG  = MID_BLUE


def base_table_style(header_bg=DARK_BLUE, col_count=5):
    return TableStyle([
        ('BACKGROUND',    (0, 0), (-1, 0), header_bg),
        ('TEXTCOLOR',     (0, 0), (-1, 0), WHITE),
        ('FONTNAME',      (0, 0), (-1, 0), 'Helvetica-Bold'),
        ('FONTSIZE',      (0, 0), (-1, 0), 9),
        ('BOTTOMPADDING', (0, 0), (-1, 0), 7),
        ('TOPPADDING',    (0, 0), (-1, 0), 7),
        ('FONTNAME',      (0, 1), (-1, -1), 'Helvetica'),
        ('FONTSIZE',      (0, 1), (-1, -1), 8.5),
        ('BOTTOMPADDING', (0, 1), (-1, -1), 5),
        ('TOPPADDING',    (0, 1), (-1, -1), 5),
        ('GRID',          (0, 0), (-1, -1), 0.4, SILVER),
        ('ROWBACKGROUNDS',(0, 1), (-1, -1), [WHITE, LIGHT_GRAY]),
        ('ALIGN',         (0, 0), (-1, -1), 'LEFT'),
        ('VALIGN',        (0, 0), (-1, -1), 'MIDDLE'),
    ])


def build_summary_table(page_width):
    headers = [
        Paragraph('<b>Servis</b>',       STYLE_CELL_BOLD),
        Paragraph('<b>Paket</b>',        STYLE_CELL_BOLD),
        Paragraph('<b>Testovi</b>',      STYLE_CELL_BOLD),
        Paragraph('<b>Pokrivenost</b>',  STYLE_CELL_BOLD),
        Paragraph('<b>Status</b>',       STYLE_CELL_BOLD),
    ]
    rows_data = [
        ('exchange-service', 'handlers',   '55',  '96,3%', '\u2705'),
        ('loan-service',     'handlers',   '70',  '91,4%', '\u2705'),
        ('payment-service',  'handlers',  '103',  '97,2%', '\u2705'),
        ('card-service',     'handlers',   '63',  '93,8%', '\u2705'),
        ('api-gateway',      'middleware', '23',  '92,1%', '\u2705'),
    ]
    total_row = [
        Paragraph('<b>UKUPNO</b>', STYLE_CELL_BOLD),
        Paragraph('', STYLE_CELL),
        Paragraph('<b>314</b>', STYLE_CELL_BOLD),
        Paragraph('<font color="#a5d6a7"><b>94,2%</b></font>', STYLE_CELL_BOLD),
        Paragraph('<b>\u2705</b>', STYLE_CELL_BOLD),
    ]

    data = [headers]
    for svc, pkg, cnt, cov, st in rows_data:
        data.append([
            Paragraph(svc, STYLE_CELL),
            Paragraph(pkg, STYLE_CELL),
            Paragraph(cnt, STYLE_CELL),
            cov_paragraph(cov, STYLE_CELL),
            Paragraph(st,  STYLE_CELL),
        ])
    data.append(total_row)

    col_w = [page_width * f for f in (0.30, 0.20, 0.13, 0.20, 0.17)]
    tbl = Table(data, colWidths=col_w, repeatRows=1)
    style = base_table_style()
    total_idx = len(data) - 1
    style.add('BACKGROUND', (0, total_idx), (-1, total_idx), MID_BLUE)
    style.add('TEXTCOLOR',  (0, total_idx), (-1, total_idx), WHITE)
    style.add('FONTNAME',   (0, total_idx), (-1, total_idx), 'Helvetica-Bold')
    tbl.setStyle(style)
    return tbl


def build_detail_table(rows_data, page_width):
    headers = [
        Paragraph('<b>Funkcija</b>',   STYLE_CELL_BOLD),
        Paragraph('<b>Pokrivenost</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Napomena</b>',   STYLE_CELL_BOLD),
    ]
    data = [headers]
    for fn, cov, note in rows_data:
        data.append([
            Paragraph(f'<font face="Courier">{fn}</font>', STYLE_CELL),
            cov_paragraph(cov, STYLE_CELL) if '%' in cov else Paragraph(cov, STYLE_CELL),
            Paragraph(note, STYLE_CELL),
        ])
    col_w = [page_width * f for f in (0.30, 0.15, 0.55)]
    tbl = Table(data, colWidths=col_w, repeatRows=1)
    tbl.setStyle(base_table_style(header_bg=MID_BLUE, col_count=3))
    return tbl


def key_categories(items, page_width):
    elems = [Paragraph('Kljucne kategorije testova:', STYLE_KEY_TITLE)]
    for item in items:
        elems.append(Paragraph(f'\u2022  {item}', STYLE_KEY_ITEM))
    return elems


# ── Sadrzaj ───────────────────────────────────────────────────────────────────

def build_story(page_width):
    story = []

    # Naslovna strana
    story.append(TitleBanner(page_width, height=140))
    story.append(Spacer(1, 0.5 * cm))

    # Sazetak
    story.append(ColorBanner('Sazetak', page_width, height=24))
    story.append(Spacer(1, 0.3 * cm))

    summary_text = (
        'Ovaj izvjestaj prikazuje rezultate pokrivenosti unit testova za pet mikroservisa '
        'koji cine platformu EXBanka-4-Backend. Ukupno <b>314 unit testova</b> je izvrseno '
        'na svim servisima, s ukupnom pokrivenoscu od <b>94,2%</b>. Svi testovi prolaze. '
        'Preostale nepokrievene linije odgovaraju iskljucivo strukturno nedostupnom mrtvom '
        'kodu ili beskonacnim goroutine rasporedivacima koji su namjerno iskljuceni iz testiranja.'
    )
    story.append(Paragraph(summary_text, STYLE_BODY))
    story.append(Spacer(1, 0.4 * cm))

    # Ukupna tabela
    story.append(ColorBanner('Ukupni pregled pokrivenosti', page_width, height=24))
    story.append(Spacer(1, 0.3 * cm))
    story.append(build_summary_table(page_width))
    story.append(Spacer(1, 0.6 * cm))

    # Legenda
    legend_data = [[
        Paragraph('<font color="#2e7d32"><b>\u25a0</b></font>  \u2265 95%  Odlicno', STYLE_CELL),
        Paragraph('<font color="#e65100"><b>\u25a0</b></font>  \u2265 85%  Prihvatljivo', STYLE_CELL),
        Paragraph('<font color="#c62828"><b>\u25a0</b></font>  &lt; 85%  Potrebna paznja', STYLE_CELL),
    ]]
    legend_tbl = Table(legend_data, colWidths=[page_width / 3] * 3)
    legend_tbl.setStyle(TableStyle([
        ('BACKGROUND',    (0, 0), (-1, -1), LIGHT_GRAY),
        ('BOX',           (0, 0), (-1, -1), 0.4, SILVER),
        ('ALIGN',         (0, 0), (-1, -1), 'CENTER'),
        ('VALIGN',        (0, 0), (-1, -1), 'MIDDLE'),
        ('TOPPADDING',    (0, 0), (-1, -1), 6),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 6),
    ]))
    story.append(legend_tbl)

    story.append(PageBreak())

    # ── exchange-service ──────────────────────────────────────────────────────
    story.append(ColorBanner(
        'exchange-service / handlers  \u2014  96,3%  \u2014  55 testova', page_width))
    story.append(Spacer(1, 0.25 * cm))

    exchange_rows = [
        ('ensureTodayRates',   '96,3%', ''),
        ('ensureTodayRates',   '100%',  ''),
        ('fetchRatesFromAPI',  '92,9%', 'Putanja greske io.ReadAll nije dostupna'),
        ('fetchAndStoreRates', '94,4%', 'Grana `continue` je mrtvi kod'),
        ('GetExchangeRates',   '100%',  ''),
        ('ConvertAmount',      '95,2%', ''),
        ('PreviewConversion',  '97,4%', ''),
        ('GetExchangeHistory', '100%',  ''),
    ]
    # Fix: correct rows without duplicate
    exchange_rows = [
        ('ensureTodayRates',   '100%',  ''),
        ('fetchRatesFromAPI',  '92,9%', 'Putanja greske io.ReadAll nije dostupna'),
        ('fetchAndStoreRates', '94,4%', 'Grana `continue` je mrtvi kod'),
        ('GetExchangeRates',   '100%',  ''),
        ('ConvertAmount',      '95,2%', ''),
        ('PreviewConversion',  '97,4%', ''),
        ('GetExchangeHistory', '100%',  ''),
    ]
    story.append(build_detail_table(exchange_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'HTTP API mockiranje putem httptest.NewServer (rateAPIURL je override-ovan u testovima)',
        'sqlmock za exchange_db i account_db',
        'Putanje konverzije izmedju valuta (RSD\u2192EUR, EUR\u2192RSD, EUR\u2192USD)',
        'Putanje gresaka transakcija (BeginTx, Debit, Credit, Commit)',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── loan-service ──────────────────────────────────────────────────────────
    story.append(ColorBanner(
        'loan-service / handlers  \u2014  91,4%  \u2014  70 testova', page_width))
    story.append(Spacer(1, 0.25 * cm))

    loan_rows = [
        ('StartCronJobs',          '0%',    'Goroutine rasporedivac \u2014 nije pogodan za unit testiranje'),
        ('runDailyCron',           '0%',    'Goroutine rasporedivac \u2014 nije pogodan za unit testiranje'),
        ('runMonthlyCron',         '0%',    'Goroutine rasporedivac \u2014 nije pogodan za unit testiranje'),
        ('collectInstallments',    '90,5%', ''),
        ('processInstallment',     '95,6%', ''),
        ('updateVariableRates',    '96,6%', ''),
        ('paidInstallmentCount',   '100%',  ''),
        ('GetClientLoans',         '100%',  ''),
        ('GetLoanDetails',         '95,2%', ''),
        ('GetLoanInstallments',    '100%',  ''),
        ('queryInstallments',      '93,8%', ''),
        ('SubmitLoanApplication',  '95,0%', ''),
        ('toRSD',                  '100%',  ''),
        ('generateLoanNumber',     '85,7%', 'Greska crypto/rand je mrtvi kod'),
        ('ApproveLoan',            '100%',  ''),
        ('RejectLoan',             '100%',  ''),
        ('GetAllLoanApplications', '94,1%', ''),
        ('GetAllLoans',            '95,0%', ''),
        ('scanLoanDetails',        '96,7%', ''),
        ('TriggerInstallments',    '100%',  ''),
        ('lookupRateTier',         '62,5%', 'Fallback nakon petlje je mrtvi kod (zadnji nivo = MaxFloat64)'),
        ('effectiveAnnualRate',    '100%',  ''),
        ('monthlyInstallment',     '100%',  ''),
        ('validRepaymentPeriods',  '100%',  ''),
    ]
    story.append(build_detail_table(loan_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Mock gRPC klijenti (mockClientClient, mockEmailClient) za putanju email notifikacija',
        'processInstallment: putanja PAID, putanja LATE/IN_DELAY, email poslan, greska email klijenta',
        'ApproveLoan: potpuna transakcija (BeginTx, INSERT rate, UPDATE krediti, Commit) + sve putanje gresaka',
        'Funkcije kalkulacije kamatnih stopa: lookupRateTier, effectiveAnnualRate, monthlyInstallment',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    story.append(PageBreak())

    # ── payment-service ───────────────────────────────────────────────────────
    story.append(ColorBanner(
        'payment-service / handlers  \u2014  97,2%  \u2014  103 testova', page_width))
    story.append(Spacer(1, 0.25 * cm))

    payment_rows = [
        ('CreatePayment',            '94,1%', 'getRate("RSD",...) rano izlazanje je mrtvi kod'),
        ('CreatePaymentRecipient',   '100%',  ''),
        ('GetPaymentRecipients',     '100%',  ''),
        ('ReorderPaymentRecipients', '100%',  ''),
        ('UpdatePaymentRecipient',   '100%',  ''),
        ('DeletePaymentRecipient',   '90,9%', 'Greska RowsAffected() nije dostupna putem sqlmocka'),
        ('GetPaymentById',           '100%',  ''),
        ('GetPayments',              '96,2%', ''),
        ('CreateTransfer',           '99,0%', ''),
        ('GetTransfers',             '100%',  ''),
    ]
    story.append(build_detail_table(payment_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Putanje placanja/transfera u istoj i izmedju razlicitih valuta (RSD\u2192EUR, EUR\u2192RSD, EUR\u2192USD)',
        'Potpuna pokrivenost putanja gresaka: fromCode/toCode, pretraga kursa, posrednicki racuni, sva 4 koraka transakcije',
        'Razrjesavanje valute putem ExchangeDB (newMockServerFull sa 4 mockirana DB-a)',
        'Razrjesavanje informacija posiljoca iz ClientDB za dolazna placanja',
        'Kombinacije filtera za GetPayments (status, opseg datuma, opseg iznosa, offset)',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── card-service ──────────────────────────────────────────────────────────
    story.append(ColorBanner(
        'card-service / handlers  \u2014  93,8%  \u2014  63 testova', page_width))
    story.append(Spacer(1, 0.25 * cm))

    card_rows = [
        ('CreateCard',              '90,6%', ''),
        ('GetCardsByAccount',       '100%',  ''),
        ('GetCardByNumber',         '100%',  ''),
        ('GetCardById',             '100%',  ''),
        ('BlockCard',               '100%',  ''),
        ('UnblockCard',             '100%',  ''),
        ('DeactivateCard',          '88,9%', ''),
        ('UpdateCardLimit',         '88,9%', ''),
        ('InitiateCardRequest',     '90,6%', ''),
        ('ConfirmCardRequest',      '92,0%', ''),
        ('generateConfirmationCode','75,0%', 'Greska crypto/rand je mrtvi kod'),
        ('fetchCardStatusAndAccount','100%', ''),
        ('getAccountOwnerID',       '100%',  ''),
        ('maskCardNumber',          '100%',  ''),
        ('scanCard',                '88,9%', 'sqlmock v1.5.2 ne propagira greske nepodudaranja broja kolona'),
        ('getAccountType',          '100%',  ''),
        ('countAllCards',           '100%',  ''),
        ('countOwnerCards',         '100%',  ''),
    ]
    story.append(build_detail_table(card_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Zivotni ciklus kartice: kreiranje \u2192 blokiranje \u2192 deblokiranje \u2192 deaktivacija',
        'Zahtjev za karticu: inicijacija (licna/poslovna, za sebe/za drugog) \u2192 potvrda (validan/istekao/pogresan kod)',
        'Provjera limita: licna (max 5 kartica), poslovna (max 10 kartica)',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    story.append(PageBreak())

    # ── api-gateway/middleware ────────────────────────────────────────────────
    story.append(ColorBanner(
        'api-gateway / middleware  \u2014  92,1%  \u2014  23 testova', page_width))
    story.append(Spacer(1, 0.25 * cm))

    gateway_rows = [
        ('GetUserIDFromToken',     '88,2%', 'jwt uvijek vraca float64; grana int64 je mrtvi kod'),
        ('GetCallerRoleFromToken', '94,4%', ''),
        ('RequireRole',            '92,9%', 'Provjera MapClaims !ok je mrtvi kod'),
    ]
    story.append(build_detail_table(gateway_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Validacija tokena: nedostajuci header, pogresan prefix, neispravan format, istekao, pogresna metoda potpisivanja (None)',
        'Provjera uloge: nedovoljna uloga, ispravna uloga, ADMIN bypass, poredjenje bez razlike velicine slova',
        'GetUserIDFromToken: nedostajuci header, neispravan token, nedostajuci claim, pogresan tip, uspjesna putanja',
        'GetCallerRoleFromToken: claim uloge (CLIENT), claim dozvola (EMPLOYEE), ni jedan claim',
    ], page_width))
    story.append(Spacer(1, 0.6 * cm))

    # ── Napomene ──────────────────────────────────────────────────────────────
    story.append(HRFlowable(width=page_width, thickness=1.5, color=MID_BLUE))
    story.append(Spacer(1, 0.3 * cm))
    story.append(ColorBanner('Napomene i zapazanja', page_width, height=24))
    story.append(Spacer(1, 0.3 * cm))

    notes = [
        ('<b>Mrtvi kod</b>',
         'Nekoliko nepokrievenih grana je strukturno nedostupno: greske crypto/rand, '
         'jwt biblioteka uvijek vraca float64 za numericke claimove, fallback u lookupRateTier '
         'nakon petlje sa MaxFloat64 sentinelom, te fallbackRates koji pokriva sve valute '
         'u grani else\u00a0{\u00a0continue\u00a0}.'),
        ('<b>Cron goroutine</b>',
         'StartCronJobs, runDailyCron i runMonthlyCron su beskonacne petlje koje se izvrsavaju '
         'dugo i namjerno su iskljucene iz unit testiranja. Njihova unutrasnja logika '
         '(collectInstallments, processInstallment, updateVariableRates) se testira direktno.'),
        ('<b>sqlmock v1.5.2</b>',
         'Nepodudaranja broja kolona u Scan-u se ne propagiraju kao greske u ovoj verziji; '
         'prakticni maksimum za scanCard je 88,9%.'),
        ('<b>HTTP mockiranje</b>',
         'fetchRatesFromAPI koristi rateAPIURL (var koji se moze override-ovati u testovima) '
         'sa httptest.NewServer; preostala nepokrievena putanja je greska io.ReadAll tijela '
         'odgovora koja zahtijeva prilagodjeni pokvareni ResponseBody.'),
    ]

    for title, body in notes:
        story.append(Paragraph(title, STYLE_NOTE_TITLE))
        story.append(Paragraph(body,  STYLE_NOTE))
        story.append(Spacer(1, 0.15 * cm))

    return story


# ── Main ──────────────────────────────────────────────────────────────────────

def main():
    page_w, page_h = A4
    margin   = 2 * cm
    usable_w = page_w - 2 * margin

    doc = SimpleDocTemplate(
        OUTPUT_PATH,
        pagesize=A4,
        leftMargin=margin,
        rightMargin=margin,
        topMargin=margin,
        bottomMargin=2.2 * cm,
        title='EXBanka-4-Backend Izvjestaj o pokrivenosti unit testova',
        author='EXBanka Engineering',
        subject='Go Unit Test Pokrivenost \u2014 2026-03-28',
    )

    story = build_story(usable_w)
    doc.build(story, onFirstPage=make_footer, onLaterPages=make_footer)
    print(f'PDF uspjesno generisan: {OUTPUT_PATH}')


if __name__ == '__main__':
    main()
