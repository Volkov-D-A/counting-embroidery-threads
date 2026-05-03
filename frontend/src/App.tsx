import {useMemo, useState} from 'react';
import './App.css';
import {
    DeleteUserPaletteEntry,
    GetPalette,
    GetTransformationSettings,
    GetTransformationSettingsPath,
    GetUserPalettePath,
    ImportThreadFile,
    ImportThreadFilePath,
    RecalculateThreadCodes,
    SaveTransformationSettings,
    SaveUserPaletteEntry,
} from '../wailsjs/go/main/App';
import {dmc, threadcalc} from '../wailsjs/go/models';

type ThreadResult = {
    code: string;
    colorName: string;
    colorHex: string;
    paletteFound: boolean;
    meters: number;
    skeins: number;
    notes: string[] | null;
};

type ImportResult = {
    cancelled: boolean;
    filePath: string;
    fileName: string;
    encoding: string;
    rowsImported: number;
    beadRowsIgnored: number;
    totalMeters: number;
    totalSkeins: number;
    skeinLengthMeters: number;
    items: ThreadResult[] | null;
    warnings: string[] | null;
};

type CodeCorrection = {
    from: string;
    to: string;
};

type PaletteEntry = {
    code: string;
    name: string;
    hex: string;
    source?: string;
};

type PaletteDraft = {
    code: string;
    name: string;
    hex: string;
};

type DescriptionTransformRule = {
    enabled: boolean;
    matchColumn: string;
    matchMode: string;
    description: string;
    stripCodePrefix: string;
    codePrefix: string;
    codeSuffix: string;
};

type TransformationSettings = {
    rules: DescriptionTransformRule[] | null;
};

type SummaryLanguage = 'ru' | 'eng';

const metersFormatter = new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
});

const compactMetersFormatter = new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
});

function formatMeters(value: number) {
    return `${metersFormatter.format(value)} м`;
}

function formatCompactMeters(value: number) {
    return compactMetersFormatter.format(value);
}

function App() {
    const [skeinLength, setSkeinLength] = useState(8);
    const [result, setResult] = useState<ImportResult | null>(null);
    const [error, setError] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [codeDrafts, setCodeDrafts] = useState<Record<string, string>>({});
    const [summaryLanguage, setSummaryLanguage] = useState<SummaryLanguage | null>(null);
    const [copyStatus, setCopyStatus] = useState('');
    const [isPaletteOpen, setIsPaletteOpen] = useState(false);
    const [palette, setPalette] = useState<PaletteEntry[]>([]);
    const [paletteSearch, setPaletteSearch] = useState('');
    const [palettePath, setPalettePath] = useState('');
    const [paletteDraft, setPaletteDraft] = useState<PaletteDraft>({code: '', name: '', hex: '#000000'});
    const [isPaletteLoading, setIsPaletteLoading] = useState(false);
    const [paletteMessage, setPaletteMessage] = useState('');
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);
    const [settings, setSettings] = useState<TransformationSettings>({rules: []});
    const [settingsPath, setSettingsPath] = useState('');
    const [isSettingsLoading, setIsSettingsLoading] = useState(false);
    const [settingsMessage, setSettingsMessage] = useState('');
    const items = result?.items ?? [];
    const warnings = result?.warnings ?? [];

    const unknownColorCount = useMemo(() => {
        return items.filter((item) => !item.paletteFound).length;
    }, [items]);

    const summaryText = useMemo(() => {
        if (!summaryLanguage) {
            return '';
        }
        return formatSummary(items, summaryLanguage);
    }, [items, summaryLanguage]);

    const filteredPalette = useMemo(() => {
        const query = paletteSearch.trim().toLowerCase();
        if (!query) {
            return palette;
        }
        return palette.filter((color) => {
            return color.code.toLowerCase().includes(query) ||
                color.name.toLowerCase().includes(query) ||
                color.hex.toLowerCase().includes(query);
        });
    }, [palette, paletteSearch]);

    async function importFile() {
        setIsLoading(true);
        setError('');
        try {
            const imported = await ImportThreadFile(skeinLength) as ImportResult;
            if (imported && !imported.cancelled) {
                setResult(normalizeResult(imported));
                setCodeDrafts({});
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsLoading(false);
        }
    }

    async function recalculate(corrections: CodeCorrection[] = [], nextSkeinLength = skeinLength) {
        if (!result) {
            return;
        }

        setIsLoading(true);
        setError('');
        try {
            const recalculated = await RecalculateThreadCodes(
                result as unknown as threadcalc.ImportResult,
                corrections as threadcalc.CodeCorrection[],
                nextSkeinLength,
            ) as ImportResult;
            setResult(normalizeResult(recalculated));
            setCodeDrafts({});
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsLoading(false);
        }
    }

    async function commitCodeCorrection(item: ThreadResult) {
        const targetCode = (codeDrafts[item.code] ?? item.code).trim();
        if (!targetCode || targetCode === item.code) {
            setCodeDrafts((drafts) => {
                const next = {...drafts};
                delete next[item.code];
                return next;
            });
            return;
        }

        await recalculate([{from: item.code, to: targetCode}]);
    }

    function updateSkeinLength(value: number) {
        setSkeinLength(Number.isFinite(value) ? value : 8);
    }

    function openSummary(language: SummaryLanguage) {
        setCopyStatus('');
        setSummaryLanguage(language);
    }

    async function copySummary() {
        if (!summaryText) {
            return;
        }

        try {
            await copyToClipboard(summaryText);
            setCopyStatus('Скопировано');
        } catch (err) {
            setCopyStatus('Не удалось скопировать');
        }
    }

    async function openPalette() {
        setIsPaletteOpen(true);
        setPaletteMessage('');
        if (palette.length || isPaletteLoading) {
            return;
        }

        await loadPalette();
    }

    async function loadPalette() {
        setIsPaletteLoading(true);
        setError('');
        try {
            const [colors, path] = await Promise.all([
                GetPalette() as Promise<PaletteEntry[]>,
                GetUserPalettePath(),
            ]);
            setPalette(colors ?? []);
            setPalettePath(path);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsPaletteLoading(false);
        }
    }

    function editPaletteEntry(color: PaletteEntry) {
        setPaletteDraft({
            code: color.code,
            name: color.name,
            hex: color.hex,
        });
        setPaletteMessage(color.source === 'user' ? 'Редактирование пользовательского цвета' : 'Будет создано пользовательское перекрытие');
    }

    function clearPaletteDraft() {
        setPaletteDraft({code: '', name: '', hex: '#000000'});
        setPaletteMessage('');
    }

    async function savePaletteEntry() {
        setIsPaletteLoading(true);
        setError('');
        try {
            await SaveUserPaletteEntry(paletteDraft as dmc.PaletteEntry);
            await loadPalette();
            setPaletteMessage('Пользовательская палитра сохранена');
            if (result) {
                await recalculate([], skeinLength);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsPaletteLoading(false);
        }
    }

    async function deletePaletteEntry(code: string) {
        setIsPaletteLoading(true);
        setError('');
        try {
            await DeleteUserPaletteEntry(code);
            await loadPalette();
            clearPaletteDraft();
            setPaletteMessage('Пользовательский цвет удален');
            if (result) {
                await recalculate([], skeinLength);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsPaletteLoading(false);
        }
    }

    async function openSettings() {
        setIsSettingsOpen(true);
        setSettingsMessage('');
        if ((settings.rules ?? []).length || isSettingsLoading) {
            return;
        }

        await loadSettings();
    }

    async function loadSettings() {
        setIsSettingsLoading(true);
        setError('');
        try {
            const [loadedSettings, path] = await Promise.all([
                GetTransformationSettings() as Promise<TransformationSettings>,
                GetTransformationSettingsPath(),
            ]);
            setSettings({rules: loadedSettings.rules ?? []});
            setSettingsPath(path);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsSettingsLoading(false);
        }
    }

    function addTransformRule() {
        setSettings((current) => ({
            rules: [
                ...(current.rules ?? []),
                {enabled: true, matchColumn: 'description', matchMode: 'equals', description: '', stripCodePrefix: '', codePrefix: '', codeSuffix: ''},
            ],
        }));
    }

    function updateTransformRule(index: number, patch: Partial<DescriptionTransformRule>) {
        setSettings((current) => ({
            rules: (current.rules ?? []).map((rule, ruleIndex) => (
                ruleIndex === index ? {...rule, ...patch} : rule
            )),
        }));
    }

    function deleteTransformRule(index: number) {
        setSettings((current) => ({
            rules: (current.rules ?? []).filter((_, ruleIndex) => ruleIndex !== index),
        }));
    }

    async function saveSettings() {
        setIsSettingsLoading(true);
        setError('');
        try {
            const saved = await SaveTransformationSettings(settings as threadcalc.TransformationSettings) as TransformationSettings;
            setSettings({rules: saved.rules ?? []});
            setSettingsMessage('Настройки сохранены');
            if (result?.filePath) {
                const imported = await ImportThreadFilePath(result.filePath, skeinLength) as ImportResult;
                setResult(normalizeResult(imported));
                setCodeDrafts({});
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setIsSettingsLoading(false);
        }
    }

    return (
        <main className="app-shell">
            <section className="toolbar" aria-label="Импорт">
                <div className="toolbar-status">
                    <div className="file-name">{result?.fileName ?? 'Файл не выбран'}</div>
                    <section className="summary" aria-label="Сводка">
                        <div className="metric">
                            <span>Цветов</span>
                            <strong>{items.length}</strong>
                        </div>
                        <div className="metric">
                            <span>Строк</span>
                            <strong>{result?.rowsImported ?? 0}</strong>
                        </div>
                        <div className="metric">
                            <span>Без палитры</span>
                            <strong>{unknownColorCount}</strong>
                        </div>
                    </section>
                </div>

                <div className="toolbar-actions">
                    <label className="field">
                        <span>Длина мотка, м</span>
                        <input
                            type="number"
                            min="0.1"
                            step="0.1"
                            value={skeinLength}
                            onChange={(event) => updateSkeinLength(Number(event.target.value))}
                            onKeyDown={(event) => {
                                if (event.key === 'Enter') {
                                    recalculate([], skeinLength);
                                }
                            }}
                        />
                    </label>
                    <button className="secondary-button" onClick={() => recalculate()} disabled={isLoading || !result}>
                        Пересчитать
                    </button>
                    <button className="secondary-button" onClick={openPalette} disabled={isPaletteLoading}>
                        Палитра
                    </button>
                    <button className="secondary-button" onClick={openSettings} disabled={isSettingsLoading}>
                        Настройки
                    </button>
                    <button className="primary-button" onClick={importFile} disabled={isLoading}>
                        {isLoading ? 'Импорт...' : 'Открыть TXT'}
                    </button>
                </div>
            </section>

            {error && <div className="alert alert-error">{error}</div>}

            {warnings.length ? (
                <section className="warnings" aria-label="Предупреждения">
                    {warnings.map((warning) => (
                        <div className="alert" key={warning}>{warning}</div>
                    ))}
                </section>
            ) : null}

            <section className="table-wrap" aria-label="Расход нитей">
                <table>
                    <thead>
                    <tr>
                        <th className="swatch-column">Цвет</th>
                        <th>Код DMC</th>
                        <th>Название</th>
                        <th className="number-column">Метраж</th>
                        <th className="number-column">Мотки</th>
                        <th>Статус</th>
                    </tr>
                    </thead>
                    <tbody>
                    {items.length ? items.map((item) => {
                        const notes = item.notes ?? [];
                        return (
                            <tr key={item.code}>
                                <td className="swatch-column">
                                    <span
                                        className={`swatch ${item.paletteFound ? '' : 'swatch-missing'}`}
                                        style={{backgroundColor: item.colorHex}}
                                    />
                                </td>
                                <td className="code-cell">
                                    <input
                                        className="code-input"
                                        value={codeDrafts[item.code] ?? item.code}
                                        disabled={isLoading}
                                        aria-label={`Код DMC ${item.code}`}
                                        onChange={(event) => {
                                            const value = event.target.value;
                                            setCodeDrafts((drafts) => ({...drafts, [item.code]: value}));
                                        }}
                                        onBlur={() => commitCodeCorrection(item)}
                                        onKeyDown={(event) => {
                                            if (event.key === 'Enter') {
                                                event.currentTarget.blur();
                                            }
                                            if (event.key === 'Escape') {
                                                setCodeDrafts((drafts) => {
                                                    const next = {...drafts};
                                                    delete next[item.code];
                                                    return next;
                                                });
                                                event.currentTarget.blur();
                                            }
                                        }}
                                    />
                                </td>
                                <td className="name-cell">{item.colorName || 'не найдено'}</td>
                                <td className="number-column">{formatMeters(item.meters)}</td>
                                <td className="number-column">{item.skeins}</td>
                                <td>
                                    <div className="notes">
                                        {notes.length ? notes.map((note) => (
                                            <span className="note" key={note}>{note}</span>
                                        )) : <span className="note note-muted">ОК</span>}
                                    </div>
                                </td>
                            </tr>
                        )
                    }) : (
                        <tr>
                            <td colSpan={6} className="empty-state">Нет данных</td>
                        </tr>
                    )}
                    </tbody>
                </table>
            </section>

            <div className="floating-summary-actions" aria-label="Компактный результат">
                <button type="button" onClick={() => openSummary('ru')} disabled={!items.length}>
                    RU
                </button>
                <button type="button" onClick={() => openSummary('eng')} disabled={!items.length}>
                    ENG
                </button>
            </div>

            {summaryLanguage && (
                <div
                    className="modal-backdrop"
                    onMouseDown={(event) => {
                        if (event.target === event.currentTarget) {
                            setSummaryLanguage(null);
                        }
                    }}
                >
                    <section className="summary-modal" role="dialog" aria-modal="true" aria-label="Компактный результат">
                        <header className="modal-header">
                            <strong>{summaryLanguage === 'ru' ? 'RU' : 'ENG'}</strong>
                            <div className="modal-actions">
                                <span className="copy-status">{copyStatus}</span>
                                <button type="button" className="secondary-button" onClick={copySummary}>
                                    Копировать
                                </button>
                                <button type="button" className="icon-button" aria-label="Закрыть" onClick={() => setSummaryLanguage(null)}>
                                    ×
                                </button>
                            </div>
                        </header>
                        <pre className="summary-text">{summaryText}</pre>
                    </section>
                </div>
            )}

            {isPaletteOpen && (
                <div
                    className="modal-backdrop"
                    onMouseDown={(event) => {
                        if (event.target === event.currentTarget) {
                            setIsPaletteOpen(false);
                        }
                    }}
                >
                    <section className="palette-modal" role="dialog" aria-modal="true" aria-label="Палитра DMC">
                        <header className="modal-header">
                            <div className="modal-title">
                                <strong>Палитра DMC</strong>
                                <span>{filteredPalette.length} из {palette.length}</span>
                            </div>
                            <div className="modal-actions">
                                <label className="palette-search">
                                    <span>Поиск</span>
                                    <input
                                        value={paletteSearch}
                                        onChange={(event) => setPaletteSearch(event.target.value)}
                                        placeholder="Код, название, HEX"
                                        autoFocus
                                    />
                                </label>
                                <button type="button" className="icon-button" aria-label="Закрыть" onClick={() => setIsPaletteOpen(false)}>
                                    ×
                                </button>
                            </div>
                        </header>
                        <section className="palette-editor" aria-label="Пользовательская палитра">
                            <label>
                                <span>Код</span>
                                <input
                                    value={paletteDraft.code}
                                    onChange={(event) => setPaletteDraft((draft) => ({...draft, code: event.target.value}))}
                                    placeholder="32"
                                />
                            </label>
                            <label>
                                <span>Название</span>
                                <input
                                    value={paletteDraft.name}
                                    onChange={(event) => setPaletteDraft((draft) => ({...draft, name: event.target.value}))}
                                    placeholder="Dark Blueberry"
                                />
                            </label>
                            <label>
                                <span>HEX</span>
                                <input
                                    value={paletteDraft.hex}
                                    onChange={(event) => setPaletteDraft((draft) => ({...draft, hex: event.target.value}))}
                                    placeholder="#4D2E8A"
                                />
                            </label>
                            <div className="palette-editor-actions">
                                <button
                                    type="button"
                                    className="primary-button"
                                    onClick={savePaletteEntry}
                                    disabled={isPaletteLoading || !paletteDraft.code.trim() || !paletteDraft.hex.trim()}
                                >
                                    Сохранить
                                </button>
                                <button type="button" className="secondary-button" onClick={clearPaletteDraft} disabled={isPaletteLoading}>
                                    Очистить
                                </button>
                            </div>
                            <div className="palette-file" title={palettePath}>
                                <span>{paletteMessage || 'Пользовательские цвета перекрывают встроенные'}</span>
                                <span>{palettePath}</span>
                            </div>
                        </section>
                        <div className="palette-table-wrap">
                            <table className="palette-table">
                                <thead>
                                <tr>
                                    <th className="swatch-column">Цвет</th>
                                    <th>Код</th>
                                    <th>Название</th>
                                    <th>HEX</th>
                                    <th>Источник</th>
                                    <th></th>
                                </tr>
                                </thead>
                                <tbody>
                                {isPaletteLoading ? (
                                    <tr>
                                        <td colSpan={6} className="empty-state">Загрузка палитры...</td>
                                    </tr>
                                ) : filteredPalette.length ? filteredPalette.map((color) => (
                                    <tr key={color.code}>
                                        <td className="swatch-column">
                                            <span className="swatch" style={{backgroundColor: color.hex}} />
                                        </td>
                                        <td className="code-cell">{color.code}</td>
                                        <td className="name-cell">{color.name}</td>
                                        <td className="hex-cell">{color.hex}</td>
                                        <td>
                                            <span className={`source-badge ${color.source === 'user' ? 'source-user' : ''}`}>
                                                {color.source === 'user' ? 'польз.' : 'встроенный'}
                                            </span>
                                        </td>
                                        <td className="row-actions">
                                            <button type="button" className="text-button" onClick={() => editPaletteEntry(color)}>
                                                Править
                                            </button>
                                            {color.source === 'user' && (
                                                <button type="button" className="text-button danger-button" onClick={() => deletePaletteEntry(color.code)}>
                                                    Удалить
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                )) : (
                                    <tr>
                                        <td colSpan={6} className="empty-state">Ничего не найдено</td>
                                    </tr>
                                )}
                                </tbody>
                            </table>
                        </div>
                    </section>
                </div>
            )}

            {isSettingsOpen && (
                <div
                    className="modal-backdrop"
                    onMouseDown={(event) => {
                        if (event.target === event.currentTarget) {
                            setIsSettingsOpen(false);
                        }
                    }}
                >
                    <section className="settings-modal" role="dialog" aria-modal="true" aria-label="Настройки преобразования">
                        <header className="modal-header">
                            <div className="modal-title">
                                <strong>Преобразование описаний</strong>
                                <span>{(settings.rules ?? []).length} правил</span>
                            </div>
                            <div className="modal-actions">
                                <button type="button" className="secondary-button" onClick={addTransformRule} disabled={isSettingsLoading}>
                                    Добавить
                                </button>
                                <button type="button" className="primary-button" onClick={saveSettings} disabled={isSettingsLoading}>
                                    Сохранить
                                </button>
                                <button type="button" className="icon-button" aria-label="Закрыть" onClick={() => setIsSettingsOpen(false)}>
                                    ×
                                </button>
                            </div>
                        </header>
                        <div className="settings-help">
                            <span>{settingsMessage || 'Если Description совпал с правилом, код меняется перед нормализацией и поиском палитры.'}</span>
                            <span title={settingsPath}>{settingsPath}</span>
                        </div>
                        <div className="settings-table-wrap">
                            <table className="settings-table">
                                <thead>
                                <tr>
                                    <th>Вкл.</th>
                                    <th>Столбец</th>
                                    <th>Режим</th>
                                    <th>Значение</th>
                                    <th>Убрать</th>
                                    <th>Префикс</th>
                                    <th>Суффикс</th>
                                    <th></th>
                                </tr>
                                </thead>
                                <tbody>
                                {isSettingsLoading ? (
                                    <tr>
                                        <td colSpan={8} className="empty-state">Загрузка настроек...</td>
                                    </tr>
                                ) : (settings.rules ?? []).length ? (settings.rules ?? []).map((rule, index) => (
                                    <tr key={index}>
                                        <td className="enabled-cell">
                                            <input
                                                type="checkbox"
                                                checked={rule.enabled}
                                                onChange={(event) => updateTransformRule(index, {enabled: event.target.checked})}
                                            />
                                        </td>
                                        <td>
                                            <select
                                                className="settings-input"
                                                value={rule.matchColumn || 'description'}
                                                onChange={(event) => updateTransformRule(index, {matchColumn: event.target.value})}
                                            >
                                                <option value="code">1-й: код</option>
                                                <option value="description">2-й: описание</option>
                                            </select>
                                        </td>
                                        <td>
                                            <select
                                                className="settings-input"
                                                value={rule.matchMode}
                                                onChange={(event) => updateTransformRule(index, {matchMode: event.target.value})}
                                            >
                                                <option value="equals">равно</option>
                                                <option value="contains">содержит</option>
                                                <option value="prefix">начинается</option>
                                                <option value="suffix">заканчивается</option>
                                            </select>
                                        </td>
                                        <td>
                                            <input
                                                className="settings-input"
                                                value={rule.description}
                                                onChange={(event) => updateTransformRule(index, {description: event.target.value})}
                                                placeholder={rule.matchColumn === 'code' ? 'DMC' : 'DMC:E'}
                                            />
                                        </td>
                                        <td>
                                            <input
                                                className="settings-input short-settings-input"
                                                value={rule.stripCodePrefix}
                                                onChange={(event) => updateTransformRule(index, {stripCodePrefix: event.target.value})}
                                                placeholder="DMC"
                                            />
                                        </td>
                                        <td>
                                            <input
                                                className="settings-input short-settings-input"
                                                value={rule.codePrefix}
                                                onChange={(event) => updateTransformRule(index, {codePrefix: event.target.value})}
                                                placeholder="E"
                                            />
                                        </td>
                                        <td>
                                            <input
                                                className="settings-input short-settings-input"
                                                value={rule.codeSuffix}
                                                onChange={(event) => updateTransformRule(index, {codeSuffix: event.target.value})}
                                            />
                                        </td>
                                        <td className="row-actions">
                                            <button type="button" className="text-button danger-button" onClick={() => deleteTransformRule(index)}>
                                                Удалить
                                            </button>
                                        </td>
                                    </tr>
                                )) : (
                                    <tr>
                                        <td colSpan={8} className="empty-state">Правил нет</td>
                                    </tr>
                                )}
                                </tbody>
                            </table>
                        </div>
                    </section>
                </div>
            )}
        </main>
    );
}

function normalizeResult(result: ImportResult): ImportResult {
    return {
        ...result,
        items: result.items ?? [],
        warnings: result.warnings ?? [],
    };
}

function formatSummary(items: ThreadResult[], language: SummaryLanguage) {
    const unit = language === 'ru' ? 'шт.' : 'pcs.';
    const meterUnit = language === 'ru' ? 'м' : 'm';

    return items
        .map((item) => `${item.code}   ${item.skeins}${unit} (${formatCompactMeters(item.meters)}${meterUnit})`)
        .join('\n');
}

async function copyToClipboard(text: string) {
    if (navigator.clipboard?.writeText) {
        try {
            await navigator.clipboard.writeText(text);
            return;
        } catch (err) {
            // Fall back to the legacy command below for stricter WebView clipboard settings.
        }
    }

    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.setAttribute('readonly', 'true');
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    document.body.appendChild(textArea);
    textArea.select();

    try {
        if (!document.execCommand('copy')) {
            throw new Error('copy command failed');
        }
    } finally {
        document.body.removeChild(textArea);
    }
}

export default App;
